package web

import (
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
)

// clkTck is the kernel's USER_HZ — CPU times in /proc are counted in these
// ticks. 100 is the near-universal value on Linux; we treat it as constant
// rather than shelling out to sysconf.
const clkTck = 100.0

// Process is one running process, as read from /proc/<pid>. It mirrors the
// columns Cockpit shows in its process table.
type Process struct {
	PID     int
	User    string // resolved from the real UID, falling back to the numeric id
	Command string // the kernel "comm" name
	State   string // process state (running, sleeping, etc.)
	Threads int
	RSS     string // resident memory, human-formatted
	CPUTime string // cumulative user+system CPU time, e.g. "12.3s"

	rssKiB int64 // kept for sorting; not rendered directly
}

type MemInfo struct {
	MemTotal float64
	MemFree  float64
}

type ProcessInfo struct {
	Processes []Process
	MemInfo   *MemInfo
}

// collectProcesses walks /proc and returns the processes using the most
// resident memory, most first. limit caps the number returned (<= 0 means all).
func collectProcesses(limit int) *ProcessInfo {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return &ProcessInfo{Processes: []Process{}, MemInfo: &MemInfo{}}
	}

	uidCache := map[string]string{}
	procs := make([]Process, 0, len(entries))
	for _, e := range entries {
		// Process directories are named by PID; everything else in /proc
		// (cpuinfo, meminfo, …) is skipped by the numeric check.
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if p, ok := readProcess(pid, uidCache); ok {
			procs = append(procs, p)
		}
	}

	sort.Slice(procs, func(i, j int) bool { return procs[i].rssKiB > procs[j].rssKiB })
	if limit > 0 && len(procs) > limit {
		procs = procs[:limit]
	}

	total, free := meminfo()
	info := &MemInfo{
		MemTotal: total,
		MemFree:  free,
	}

	return &ProcessInfo{
		Processes: procs,
		MemInfo:   info,
	}
}

// readProcess reads one process. It returns ok=false if the process vanished
// between the readdir and now (a normal race), so callers can just skip it.
func readProcess(pid int, uidCache map[string]string) (Process, bool) {
	dir := "/proc/" + strconv.Itoa(pid)

	status, err := os.ReadFile(dir + "/status")
	if err != nil {
		return Process{}, false
	}

	p := Process{PID: pid}
	for _, line := range strings.Split(string(status), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch key {
		case "Name":
			p.Command = val
		case "State":
			p.State = stateLabel(val)
		case "Threads":
			p.Threads, _ = strconv.Atoi(val)
		case "VmRSS":
			// "12345 kB"
			if f := strings.Fields(val); len(f) > 0 {
				p.rssKiB, _ = strconv.ParseInt(f[0], 10, 64)
			}
		case "Uid":
			// "real effective saved fs" — the first is the real UID.
			if f := strings.Fields(val); len(f) > 0 {
				p.User = resolveUser(f[0], uidCache)
			}
		}
	}

	p.RSS = humanKiB(p.rssKiB)
	p.CPUTime = readCPUTime(dir)
	return p, true
}

// readCPUTime returns cumulative user+system CPU time from /proc/<pid>/stat.
func readCPUTime(dir string) string {
	b, err := os.ReadFile(dir + "/stat")
	if err != nil {
		return "—"
	}
	// The "comm" field is wrapped in parens and may itself contain spaces or
	// parens, so we anchor parsing after the final ')'. After it, the fields
	// are: state ppid ... where utime/stime are stat fields 14/15 — i.e. the
	// 12th and 13th tokens once state is index 0.
	s := string(b)
	rparen := strings.LastIndexByte(s, ')')
	if rparen < 0 {
		return "—"
	}
	fields := strings.Fields(s[rparen+1:])
	if len(fields) < 13 {
		return "—"
	}
	utime, _ := strconv.ParseFloat(fields[11], 64)
	stime, _ := strconv.ParseFloat(fields[12], 64)
	secs := (utime + stime) / clkTck
	return strconv.FormatFloat(secs, 'f', 1, 64) + "s"
}

// resolveUser maps a numeric UID to a username, caching lookups. If the user
// can't be resolved it returns the raw UID.
func resolveUser(uid string, cache map[string]string) string {
	if name, ok := cache[uid]; ok {
		return name
	}
	name := uid
	if u, err := user.LookupId(uid); err == nil {
		name = u.Username
	}
	cache[uid] = name
	return name
}

// stateLabel turns the raw status "State" value ("S (sleeping)") into a plain
// word. It falls back to the raw value for anything unexpected.
func stateLabel(raw string) string {
	code := raw
	if i := strings.IndexByte(raw, ' '); i > 0 {
		code = raw[:i]
	}
	switch code {
	case "R":
		return "running"
	case "S":
		return "sleeping"
	case "D":
		return "uninterruptible"
	case "Z":
		return "zombie"
	case "T":
		return "stopped"
	case "t":
		return "tracing-stop"
	case "I":
		return "idle"
	case "X", "x":
		return "dead"
	default:
		return raw
	}
}

// humanKiB formats a KiB count as KiB / MiB / GiB.
func humanKiB(kib int64) string {
	switch {
	case kib >= 1024*1024:
		return strconv.FormatFloat(float64(kib)/1024/1024, 'f', 1, 64) + " GiB"
	case kib >= 1024:
		return strconv.FormatFloat(float64(kib)/1024, 'f', 1, 64) + " MiB"
	default:
		return strconv.FormatInt(kib, 10) + " KiB"
	}
}

// meminfo returns MemTotal and MemAvailable from /proc/meminfo in GiB.
func meminfo() (total, free float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		gib := kb / 1024 / 1024
		switch fields[0] {
		case "MemTotal:":
			total = gib
		case "MemAvailable:":
			free = gib
		}
	}
	return
}
