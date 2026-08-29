package web

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// readCaddyfile reads the Caddy configuration file from /etc/caddy/Caddyfile
func readCaddyfile() string {
	data, err := os.ReadFile("/etc/caddy/Caddyfile")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such file or directory") {
			return "Caddyfile not present: etc/caddy/Caddyfile\nInstall Caddy and Add Caddyfile."
		}
		fmt.Println(err.Error())
		return "Error reading Caddyfile: " + err.Error()
	}
	return string(data)
}

// quadletSourceExtensions are the file extensions Quadlet's systemd
// generator recognizes as unit source files.
var quadletSourceExtensions = []string{".container", ".volume", ".network", ".pod", ".kube", ".build", ".image"}

// isQuadletSource reports whether path looks like a Quadlet source file, based on its extension.
func isQuadletSource(path string) bool {
	return slices.Contains(quadletSourceExtensions, filepath.Ext(path))
}

// sourcePath asks systemd for the file a generated unit came from, via the
// SourcePath property. Empty if the unit isn't generated or doesn't exist.
func sourcePath(unit string) string {
	out, err := exec.Command("systemctl", "show", unit, "--property=SourcePath").Output()
	if err != nil {
		return ""
	}
	_, val, ok := strings.Cut(strings.TrimSpace(string(out)), "=")
	if !ok {
		return ""
	}
	return val
}

// quadletUnits returns every systemd service that was generated from a
// Quadlet source file, mapped to that source file's path.
func quadletUnits() map[string]string {
	out, err := exec.Command("systemctl", "list-unit-files", "--type=service", "--no-legend", "--no-pager").Output()
	if err != nil {
		return nil
	}

	units := make(map[string]string)
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "generated" {
			continue
		}
		if source := sourcePath(fields[0]); isQuadletSource(source) {
			units[fields[0]] = source
		}
	}
	return units
}

// readUnitFiles reads every Quadlet unit's source file, as found via
// quadletUnits, and returns them sorted alphabetically by filename.
func readUnitFiles() []struct{ Name, Content string } {
	var containers []struct{ Name, Content string }

	for _, source := range quadletUnits() {
		data, err := os.ReadFile(source)
		if err != nil {
			continue
		}
		containers = append(containers, struct{ Name, Content string }{
			Name:    filepath.Base(source),
			Content: string(data),
		})
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers
}

// runningQuadletServices returns the systemd service names for Quadlet
// units that are currently running, by cross-referencing quadletUnits
// against systemctl's list of running services.
func runningQuadletServices() []string {
	configured := quadletUnits()
	if len(configured) == 0 {
		return nil
	}

	out, err := exec.Command("systemctl", "list-units", "--type=service", "--state=running", "--plain", "--no-legend", "--no-pager").Output()
	if err != nil {
		return nil
	}

	var running []string
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if _, ok := configured[fields[0]]; ok {
			running = append(running, fields[0])
		}
	}

	sort.Strings(running)
	return running
}

// readServiceLogs returns the last n lines of a systemd unit's journal.
func readServiceLogs(service string, n int) string {
	out, err := exec.Command("journalctl", "-u", service, "-n", strconv.Itoa(n), "--no-pager", "--output=short-iso").CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error reading logs for %s: %s\n%s", service, err, out)
	}
	return string(out)
}

// ServiceStatus is a systemd unit's live status, as shown above its log tail.
type ServiceStatus struct {
	ActiveState string // "active", "failed"
	SubState    string // "running", "dead"
	PID         string
	Since       string // human-readable duration since it entered ActiveState
	Restarts    string
}

// systemdTimestampLayout matches the default format of systemctl show's
// ActiveEnterTimestamp, e.g. "Wed 2026-08-27 10:15:32 UTC".
const systemdTimestampLayout = "Mon 2006-01-02 15:04:05 MST"

// readServiceStatus reads a systemd unit's current status via `systemctl
// show`. Properties come back as KEY=VALUE lines - not necessarily in the
// order requested, so they're parsed into a map by name
func readServiceStatus(service string) ServiceStatus {
	out, err := exec.Command(
		"systemctl", "show", service,
		"--property=ActiveState,SubState,MainPID,ActiveEnterTimestamp,NRestarts",
	).Output()
	if err != nil {
		return ServiceStatus{ActiveState: "unknown"}
	}

	props := make(map[string]string)
	for line := range strings.SplitSeq(string(out), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if ok {
			props[key] = val
		}
	}

	since := "unknown"
	if t, err := time.Parse(systemdTimestampLayout, props["ActiveEnterTimestamp"]); err == nil {
		since = formatDuration(time.Since(t))
	}

	return ServiceStatus{
		ActiveState: props["ActiveState"],
		SubState:    props["SubState"],
		PID:         props["MainPID"],
		Since:       since,
		Restarts:    props["NRestarts"],
	}
}

// readWebTree searches /var/www (the haystack) for index.html files (the
// needle) and returns the pruned path down to each one found.
func readWebTree() []*FileNode {
	return findIndexNodes("/var/www")
}

// findIndexNodes returns the FileNode(s) representing the nearest
// index.html(s) beneath dir. If dir directly contains an index.html, that's
// the answer for this branch and it returns immediately.
func findIndexNodes(dir string) []*FileNode {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if !e.IsDir() && e.Name() == "index.html" {
			return []*FileNode{{Name: "index.html"}}
		}
	}

	var subdirs []string
	for _, e := range entries {
		if e.IsDir() {
			subdirs = append(subdirs, e.Name())
		}
	}
	sort.Strings(subdirs)

	var nodes []*FileNode
	for _, sub := range subdirs {
		if children := findIndexNodes(filepath.Join(dir, sub)); children != nil {
			nodes = append(nodes, &FileNode{Name: sub, IsDir: true, Children: children})
		}
	}
	return nodes
}
