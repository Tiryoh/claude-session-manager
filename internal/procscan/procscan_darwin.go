//go:build darwin

package procscan

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Scan lists running "claude" processes via `ps` (there's no /proc on
// macOS), extracting --resume session ids from argv, then resolves their
// cwd via `lsof` (ps alone can't report a process's working directory).
// ok is false only if `ps` itself fails to run.
func Scan() (Snapshot, bool) {
	out, err := exec.Command("ps", "-axo", "pid=,comm=,args=").Output()
	if err != nil {
		return Snapshot{}, false
	}

	snap := Snapshot{SessionIDs: map[string]bool{}, Cwds: map[string]bool{}}
	var pids []string
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || filepath.Base(fields[1]) != "claude" {
			continue
		}
		pids = append(pids, fields[0])
		args := fields[2:]
		for i, a := range args {
			if a == "--resume" && i+1 < len(args) {
				snap.SessionIDs[args[i+1]] = true
			}
		}
	}
	if len(pids) == 0 {
		return snap, true
	}

	for _, cwd := range lsofCwds(pids) {
		snap.Cwds[cwd] = true
	}
	return snap, true
}

// lsofCwds resolves each pid's current working directory via `lsof -d cwd`.
// A failure here (missing lsof, permission) just means no cwd-based
// matches -- the SessionIDs already collected from argv are unaffected.
func lsofCwds(pids []string) []string {
	out, err := exec.Command("lsof", "-a", "-d", "cwd", "-p", strings.Join(pids, ","), "-Fn").Output()
	if err != nil {
		return nil
	}
	var cwds []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			cwds = append(cwds, line[1:])
		}
	}
	return cwds
}
