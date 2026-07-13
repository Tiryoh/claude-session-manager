//go:build linux

package procscan

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Scan reads /proc for every running "claude" process, correlating by cwd
// (via the /proc/<pid>/cwd symlink) and by --resume <session-id> argv (via
// /proc/<pid>/cmdline). ok is false only if /proc itself can't be listed;
// a single process disappearing mid-scan or being unreadable (permission,
// short-lived pid) is skipped silently -- this is a best-effort liveness
// hint, never a hard requirement.
func Scan() (Snapshot, bool) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return Snapshot{}, false
	}

	snap := Snapshot{SessionIDs: map[string]bool{}, Cwds: map[string]bool{}}
	for _, e := range entries {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		dir := filepath.Join("/proc", e.Name())

		comm, err := os.ReadFile(filepath.Join(dir, "comm"))
		if err != nil || filepath.Base(strings.TrimSpace(string(comm))) != "claude" {
			continue
		}

		if cwd, err := os.Readlink(filepath.Join(dir, "cwd")); err == nil {
			snap.Cwds[cwd] = true
		}

		if cmdline, err := os.ReadFile(filepath.Join(dir, "cmdline")); err == nil {
			args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
			for i, a := range args {
				if a == "--resume" && i+1 < len(args) {
					snap.SessionIDs[args[i+1]] = true
				}
			}
		}
	}
	return snap, true
}
