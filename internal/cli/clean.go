package cli

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/claudedir"
	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// RunClean removes active-session entries that are no longer restorable:
// their transcript file is gone, or they've been idle longer than
// olderThan. An idle-but-transcript-present entry is a power-loss survivor
// -- exactly what csm exists to let the user restore -- so it is kept
// regardless of age until it crosses olderThan.
func RunClean(w io.Writer, p registry.Paths, claudeDir string, olderThan time.Duration, dryRun bool, now time.Time) error {
	active, _, err := registry.LoadActive(p)
	if err != nil {
		return err
	}

	var toRemove []string
	for id, s := range active.Sessions {
		if !claudedir.TranscriptExists(claudeDir, s.Cwd, s.SessionID) {
			toRemove = append(toRemove, id)
			continue
		}
		if now.Sub(s.LastSeen) > olderThan {
			toRemove = append(toRemove, id)
		}
	}
	sort.Strings(toRemove)

	if len(toRemove) == 0 {
		fmt.Fprintln(w, "nothing to clean")
		return nil
	}
	for _, id := range toRemove {
		s := active.Sessions[id]
		verb := "would remove"
		if !dryRun {
			verb = "removed"
		}
		fmt.Fprintf(w, "%s %s  %s  (%s)\n", verb, shortID(id), s.Label, s.Cwd)
	}
	if dryRun {
		return nil
	}

	return registry.MutateActive(p, func(reg *registry.ActiveRegistry) {
		for _, id := range toRemove {
			delete(reg.Sessions, id)
		}
	})
}
