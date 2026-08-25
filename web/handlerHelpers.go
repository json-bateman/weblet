package web

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// readCaddyfile reads the Caddy configuration file from /etc/caddy/Caddyfile
func readCaddyfile() string {
	data, err := os.ReadFile("/etc/caddy/Caddyfile")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such file or directory") {
			return "Caddyfile not present in etc/caddy/Caddyfile\nInstall Caddy and Add Caddyfile."
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

// readWebTree returns the /var/www directory tree, with directories listed
// before files and each level sorted alphabetically.
func readWebTree() []*FileNode {
	return readDirTree("/var/www")
}

// readDirTree recursively reads dir into FileNodes. File contents are loaded
// eagerly; unreadable files get empty content rather than aborting the walk.
func readDirTree(dir string) []*FileNode {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	nodes := make([]*FileNode, 0, len(entries))
	for _, e := range entries {
		node := &FileNode{Name: e.Name(), IsDir: e.IsDir()}
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			node.Children = readDirTree(full)
		}
		nodes = append(nodes, node)
	}

	// Directories first, then alphabetical within each group.
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes
}
