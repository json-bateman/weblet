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

// readUnitFiles reads all .container files from /etc/containers/systemd/ and returns them sorted
func readUnitFiles() []struct{ Name, Content string } {
	var containers []struct{ Name, Content string }
	dir := "/etc/containers/systemd"

	entries, err := os.ReadDir(dir)
	if err != nil {
		return containers
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := dir + "/" + entry.Name()
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			containers = append(containers, struct{ Name, Content string }{
				Name:    entry.Name(),
				Content: string(data),
			})
		}
	}

	// Sort alphabetically by name
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers
}

// runningQuadletServices returns the systemd service names for Quadlet
// container units (/etc/containers/systemd/*.container) that are currently
// running, by cross-referencing configured units against systemctl's list
// of running services.
func runningQuadletServices() []string {
	configured := make(map[string]bool)
	for _, c := range readUnitFiles() {
		name := strings.TrimSuffix(c.Name, filepath.Ext(c.Name))
		configured[name+".service"] = true
	}
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
		if configured[fields[0]] {
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

// readWebTree searches each top-level entry under /var/www (the haystack)
// for its nearest index.html (the needle), and returns the pruned path down
// to it - directories with a single child each, ending in the index.html
// file. Site directories with no index.html anywhere beneath them are
// omitted.
func readWebTree() []*FileNode {
	const root = "/var/www"

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	nodes := make([]*FileNode, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			if matches := shortestIndexPaths(filepath.Join(root, e.Name())); matches != nil {
				nodes = append(nodes, &FileNode{Name: e.Name(), IsDir: true, Children: matches})
			}
		} else if e.Name() == "index.html" {
			nodes = append(nodes, &FileNode{Name: e.Name()})
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes
}

// shortestIndexPaths breadth-first searches dir level by level and returns
// the chain of FileNodes down to every index.html found at the shallowest
// depth that has one. Ties at the same depth (index.html files under
// different sibling subdirectories) are all included rather than just the
// first one encountered. Returns nil if dir has no index.html anywhere
// beneath it.
func shortestIndexPaths(dir string) []*FileNode {
	type queued struct {
		path  string
		chain []string
	}

	queue := []queued{{path: dir}}
	for len(queue) > 0 {
		var matches []*FileNode
		var next []queued

		for _, cur := range queue {
			entries, err := os.ReadDir(cur.path)
			if err != nil {
				continue
			}

			hasIndex := false
			var subdirs []string
			for _, e := range entries {
				if !e.IsDir() && e.Name() == "index.html" {
					hasIndex = true
				} else if e.IsDir() {
					subdirs = append(subdirs, e.Name())
				}
			}

			if hasIndex {
				leaf := &FileNode{Name: "index.html"}
				for _, name := range slices.Backward(cur.chain) {
					leaf = &FileNode{Name: name, IsDir: true, Children: []*FileNode{leaf}}
				}
				matches = append(matches, leaf)
				continue
			}

			sort.Strings(subdirs)
			for _, sub := range subdirs {
				next = append(next, queued{
					path:  filepath.Join(cur.path, sub),
					chain: append(append([]string{}, cur.chain...), sub),
				})
			}
		}

		if len(matches) > 0 {
			return matches
		}
		queue = next
	}

	return nil
}
