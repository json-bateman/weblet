package web

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ServerInfo is the live host information rendered into the page. Everything is
// read at request time — the hostname from the OS, OS/kernel from /etc and
// /proc, and the rest from the Go runtime and /proc — so it reflects whatever
// machine the binary is running on.
type ServerInfo struct {
	Name       string
	OS         string
	Kernel     string
	Arch       string
	CPUs       int
	GoVersion  string
	Uptime     string
	Now        string
	Goroutines int
}

// collectServerInfo gathers a fresh snapshot of host information.
func collectServerInfo() ServerInfo {
	name, err := os.Hostname()
	if err != nil {
		name = "unknown-host"
	}
	return ServerInfo{
		Name:       name,
		OS:         osPrettyName(),
		Kernel:     kernelRelease(),
		Arch:       runtime.GOARCH,
		CPUs:       runtime.NumCPU(),
		GoVersion:  runtime.Version(),
		Uptime:     uptime(),
		Now:        time.Now().UTC().Format("15:04:05 MST"),
		Goroutines: runtime.NumGoroutine(),
	}
}

// osPrettyName returns PRETTY_NAME from /etc/os-release (e.g. "CentOS Stream
// 10"), falling back to the OS family name off Linux or if unreadable.
func osPrettyName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	for _, line := range strings.Split(string(data), "\n") {
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
