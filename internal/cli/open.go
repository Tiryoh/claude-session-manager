package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/claudedir"
	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// ExecFunc replaces the current process with `claude --resume <id>` (or
// similar) run from dir. Production code supplies one backed by
// os.Chdir + syscall.Exec; tests supply a fake that records its arguments.
type ExecFunc func(dir string, argv []string) error

// entry is a saved item RunOpen can present in its picker: either a
// bookmark or one session inside a group.
type entry struct {
	label   string
	session registry.Session
}

// RunOpen resumes a saved bookmark or group session. If name is empty (or
// ambiguous with multiple candidates), the user is prompted interactively;
// a name that matches exactly one bookmark or group skips the picker.
func RunOpen(w io.Writer, r *bufio.Reader, p registry.Paths, claudeDir, name string, fork, isTTY bool, exec ExecFunc, now time.Time) error {
	if !isTTY {
		return fmt.Errorf("csm open requires an interactive terminal")
	}
	saved, _, err := registry.LoadSaved(p)
	if err != nil {
		return err
	}

	chosen, err := resolveEntry(w, r, saved, name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(chosen.session.Cwd); err != nil {
		return fmt.Errorf("saved directory %q no longer exists: %w", chosen.session.Cwd, err)
	}
	if !claudedir.TranscriptExists(claudeDir, chosen.session.Cwd, chosen.session.SessionID) {
		ans, err := Prompt(w, r, fmt.Sprintf("warning: transcript for %q not found, resume will likely fail; continue? [y/N]: ", chosen.label))
		if err != nil {
			return err
		}
		if ans != "y" && ans != "Y" {
			return fmt.Errorf("aborted")
		}
	}

	argv := []string{"claude", "--resume", chosen.session.SessionID}
	if fork {
		argv = append(argv, "--fork-session")
	}
	return exec(chosen.session.Cwd, argv)
}

// resolveEntry finds the entry to open. An exact bookmark-name match wins;
// otherwise if name matches a group with exactly one session, that session
// is used; otherwise the user is prompted from the full flattened list
// (bookmarks first, then each group's sessions).
func resolveEntry(w io.Writer, r *bufio.Reader, saved *registry.SavedRegistry, name string) (entry, error) {
	if name != "" {
		if b, ok := saved.Bookmarks[name]; ok {
			return entry{label: name, session: b.Session}, nil
		}
		if g, ok := saved.Groups[name]; ok {
			if len(g.Sessions) == 1 {
				return entry{label: name, session: g.Sessions[0]}, nil
			}
			return pickFromGroup(w, r, name, g)
		}
		return entry{}, fmt.Errorf("no bookmark or group named %q", name)
	}

	entries := flattenAll(saved)
	if len(entries) == 0 {
		return entry{}, fmt.Errorf("nothing saved yet -- run 'csm save' or 'csm bookmark' first")
	}
	labels := make([]string, len(entries))
	for i, e := range entries {
		labels[i] = e.label
	}
	idx, err := SelectOne(w, r, "Select a session to open: ", labels)
	if err != nil {
		return entry{}, err
	}
	return entries[idx], nil
}

func pickFromGroup(w io.Writer, r *bufio.Reader, groupName string, g registry.Group) (entry, error) {
	labels := make([]string, len(g.Sessions))
	for i, s := range g.Sessions {
		labels[i] = fmt.Sprintf("%s  %s  (%s)", shortID(s.SessionID), s.Label, s.Cwd)
	}
	idx, err := SelectOne(w, r, fmt.Sprintf("Select a session from group %q: ", groupName), labels)
	if err != nil {
		return entry{}, err
	}
	return entry{label: labels[idx], session: g.Sessions[idx]}, nil
}

func flattenAll(saved *registry.SavedRegistry) []entry {
	var entries []entry
	names := make([]string, 0, len(saved.Bookmarks))
	for n := range saved.Bookmarks {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b := saved.Bookmarks[n]
		entries = append(entries, entry{label: fmt.Sprintf("[bookmark] %s  %s  (%s)", n, b.Label, b.Cwd), session: b.Session})
	}

	gnames := make([]string, 0, len(saved.Groups))
	for n := range saved.Groups {
		gnames = append(gnames, n)
	}
	sort.Strings(gnames)
	for _, n := range gnames {
		g := saved.Groups[n]
		for _, s := range g.Sessions {
			entries = append(entries, entry{label: fmt.Sprintf("[%s] %s  %s  (%s)", n, shortID(s.SessionID), s.Label, s.Cwd), session: s})
		}
	}
	return entries
}
