package web

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// DiskInfo mirrors MemInfo's shape so the homepage can render it with the
// same progress-bar treatment.
type DiskInfo struct {
	Total float64 // GiB
	Free  float64 // GiB
}

// diskUsage sums capacity and free space across every real, distinct
// disk-backed filesystem: this reads /proc/mounts to find them all.
func diskUsage() DiskInfo {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return DiskInfo{}
	}

	seen := make(map[string]bool) // dedupes bind mounts of the same device
	var totalKiB, freeKiB uint64

	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		device, mountpoint := fields[0], unescapeMount(fields[1])

		// Only devices under /dev/ are real disks; this naturally excludes
		// tmpfs, proc, sysfs, cgroup, overlay, nfs, and other pseudo/network
		// filesystems without needing to enumerate their fstypes.
		if !strings.HasPrefix(device, "/dev/") || seen[device] {
			continue
		}
		seen[device] = true

		var stat unix.Statfs_t
		if err := unix.Statfs(mountpoint, &stat); err != nil {
			continue
		}

		blockSize := uint64(stat.Bsize)
		totalKiB += (stat.Blocks * blockSize) / 1024
		freeKiB += (stat.Bavail * blockSize) / 1024
	}

	return DiskInfo{
		Total: float64(totalKiB) / 1024 / 1024,
		Free:  float64(freeKiB) / 1024 / 1024,
	}
}

// unescapeMount decodes the octal escapes /proc/mounts uses for spaces and
// other special characters in mount paths (e.g. "\040" -> " ").
func unescapeMount(path string) string {
	return strings.ReplaceAll(path, `\040`, " ")
}
