package cli

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// RunSave snapshots the current active sessions into a named group. With
// all=true every active session is saved without prompting (used for cron:
// `csm save --all --force autosave`); otherwise the user is prompted to pick
// a subset. Saving over an existing name requires force=true.
func RunSave(w io.Writer, r *bufio.Reader, p registry.Paths, name string, all, force bool, now time.Time) error {
	active, _, err := registry.LoadActive(p)
	if err != nil {
		return err
	}
	ids := sortedSessionIDsAsc(active.Sessions)
	if len(ids) == 0 {
		return fmt.Errorf("no active sessions to save")
	}
	if name == "" {
		name = "saved-" + now.Format("2006-01-02-150405")
	}

	var chosen []int
	if all {
		chosen = make([]int, len(ids))
		for i := range ids {
			chosen[i] = i
		}
	} else {
		labels := make([]string, len(ids))
		for i, id := range ids {
			s := active.Sessions[id]
			labels[i] = fmt.Sprintf("%s  %s  (%s)", shortID(id), s.Label, s.Cwd)
		}
		chosen, err = SelectMany(w, r, "Select sessions to save (e.g. 1,3 or 'all'): ", labels)
		if err != nil {
			return err
		}
	}

	sessions := make([]registry.Session, 0, len(chosen))
	for _, i := range chosen {
		sessions = append(sessions, active.Sessions[ids[i]])
	}

	// saveErr carries a business-logic rejection (duplicate name) out of the
	// mutate closure; it's distinct from mutateErr (an I/O failure) so the
	// duplicate-name case is never masked by MutateSaved's own nil return.
	var saveErr error
	mutateErr := registry.MutateSaved(p, func(reg *registry.SavedRegistry) {
		if _, exists := reg.Groups[name]; exists && !force {
			saveErr = fmt.Errorf("group %q already exists; use --force to overwrite", name)
			return
		}
		reg.Groups[name] = registry.Group{SavedAt: now, Sessions: sessions}
	})
	if saveErr != nil {
		return saveErr
	}
	return mutateErr
}

// RunBookmark snapshots a single active session, chosen interactively, into
// a named bookmark.
func RunBookmark(w io.Writer, r *bufio.Reader, p registry.Paths, name string, now time.Time) error {
	active, _, err := registry.LoadActive(p)
	if err != nil {
		return err
	}
	ids := sortedSessionIDsAsc(active.Sessions)
	if len(ids) == 0 {
		return fmt.Errorf("no active sessions to bookmark")
	}

	labels := make([]string, len(ids))
	for i, id := range ids {
		s := active.Sessions[id]
		labels[i] = fmt.Sprintf("%s  %s  (%s)", shortID(id), s.Label, s.Cwd)
	}
	idx, err := SelectOne(w, r, "Select a session to bookmark: ", labels)
	if err != nil {
		return err
	}
	chosen := active.Sessions[ids[idx]]

	if name == "" {
		name = baseName(chosen.Cwd)
	}
	return registry.MutateSaved(p, func(reg *registry.SavedRegistry) {
		reg.Bookmarks[name] = registry.Bookmark{Session: chosen, SavedAt: now}
	})
}

func sortedSessionIDsAsc(sessions map[string]registry.Session) []string {
	ids := make([]string, 0, len(sessions))
	for id := range sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
