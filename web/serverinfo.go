package web

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Static host facts
var (
	Name      string
	Kernel    string
	Osrelease string
	Arch      string
	GoVersion string
	NumCPU    int
)

func init() {
	name, err := os.Hostname()
	if err != nil {
		name = "unknown-host"
	}
	Name = name
	Kernel = kernelRelease()
	Osrelease = osPrettyName()
	Arch = runtime.GOARCH
	GoVersion = runtime.Version()
	NumCPU = runtime.NumCPU()
}

// ServerInfo is the live host information rendered into the page. The static
// fields (Name, OS, Kernel, Arch, GoVersion, CPUs) are captured once at
// startup; Uptime, Now, Goroutines, CPUPercent, CoreUsage, and MemInfo are
// re-read on every poll since they actually change over time.
type ServerInfo struct {
	Name       string
	OS         string
	Kernel     string
	Arch       string
	GoVersion  string
	CPUs       int
	Uptime     string
	Now        string
	Goroutines int
	CPUPercent float64
	CoreUsage  []CoreUsage
	MemInfo    *MemInfo
	DiskInfo   DiskInfo
}

// collectServerInfo gathers a fresh snapshot of the host's state
func collectServerInfo() ServerInfo {
	total, free := meminfo()
	return ServerInfo{
		Name:       Name,
		OS:         Osrelease,
		Kernel:     Kernel,
		Arch:       Arch,
		GoVersion:  GoVersion,
		CPUs:       NumCPU,
		Uptime:     uptime(),
		Now:        time.Now().UTC().Format("15:04:05 MST"),
		Goroutines: runtime.NumGoroutine(),
		CPUPercent: cpuUsagePercent(),
		CoreUsage:  cpuCoreUsagePercents(),
		MemInfo:    &MemInfo{MemTotal: total, MemFree: free},
		DiskInfo:   diskUsage(),
	}
}

// osPrettyName returns PRETTY_NAME from /etc/os-release (e.g. "CentOS Stream
// 10"), falling back to the OS family name off Linux or if unreadable.
func osPrettyName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(v, `"`)
		}
	}
	return runtime.GOOS
}

// kernelRelease returns the running kernel version from /proc.
func kernelRelease() string {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// uptime reads /proc/uptime and returns a human-readable duration.
func uptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "unknown"
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "unknown"
	}
	d := time.Duration(secs) * time.Second
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
	case hours > 0:
		return strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
	default:
		return strconv.Itoa(mins) + "m"
	}
}
