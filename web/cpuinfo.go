package web

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// cpuSample is one snapshot of a /proc/stat "cpu*" line, in jiffies.
type cpuSample struct {
	idle, total float64
}

// CoreUsage is one CPU core's utilization, as shown per-core on the homepage.
type CoreUsage struct {
	Name    string // "cpu0", "cpu1", ...
	Percent float64
}

var (
	cpuMu   sync.Mutex
	cpuPrev map[string]cpuSample // keyed by /proc/stat line name: "cpu", "cpu0", "cpu1", ...
)

// cpuUsagePercent returns aggregate CPU utilization since the previous call.
func cpuUsagePercent() float64 {
	return usageDelta("cpu", readCPUSamples())
}

// cpuCoreUsagePercents returns per-core utilization since the previous call,
// ordered cpu0, cpu1, .... Utilization is a delta between two /proc/stat
// samples — a single read can't give a percentage, since the fields are
// cumulative jiffie counts since boot — so the first call in the process's
// lifetime returns 0 for every core, having nothing to diff against yet.
func cpuCoreUsagePercents() []CoreUsage {
	samples := readCPUSamples()

	var names []string
	for name := range samples {
		if name != "cpu" {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		ni, _ := strconv.Atoi(strings.TrimPrefix(names[i], "cpu"))
		nj, _ := strconv.Atoi(strings.TrimPrefix(names[j], "cpu"))
		return ni < nj
	})

	cores := make([]CoreUsage, 0, len(names))
	for _, name := range names {
		cores = append(cores, CoreUsage{Name: name, Percent: usageDelta(name, samples)})
	}
	return cores
}

// usageDelta computes utilization for one /proc/stat line name against the
// previously stored sample for that same name, then stores the new sample.
func usageDelta(name string, samples map[string]cpuSample) float64 {
	sample, ok := samples[name]
	if !ok {
		return 0
	}

	cpuMu.Lock()
	if cpuPrev == nil {
		cpuPrev = make(map[string]cpuSample)
	}
	prev, hadPrev := cpuPrev[name]
	cpuPrev[name] = sample
	cpuMu.Unlock()

	if !hadPrev {
		return 0
	}

	totalDelta := sample.total - prev.total
	if totalDelta <= 0 {
		return 0
	}
	idleDelta := sample.idle - prev.idle

	usage := (1 - idleDelta/totalDelta) * 100
	switch {
	case usage < 0:
		return 0
	case usage > 100:
		return 100
	default:
		return usage
	}
}

// readCPUSamples reads every "cpu*" line from /proc/stat — the aggregate
// "cpu" line and one "cpuN" line per core — each with fields: user, nice,
// system, idle, iowait, irq, softirq, steal, guest, guest_nice, in jiffies.
func readCPUSamples() map[string]cpuSample {
	samples := make(map[string]cpuSample)

	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return samples
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}

		var sample cpuSample
		for i, f := range fields[1:] {
			v, err := strconv.ParseFloat(f, 64)
			if err != nil {
				continue
			}
			sample.total += v
			// idle (3) and iowait (4) both count as non-busy time.
			if i == 3 || i == 4 {
				sample.idle += v
			}
		}
		samples[fields[0]] = sample
	}

	return samples
}
