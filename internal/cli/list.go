package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// RunList prints the active sessions, saved groups, and bookmarks in p.
// With jsonOut it prints one JSON object instead of the human-readable form.
func RunList(w io.Writer, p registry.Paths, jsonOut bool, now time.Time) error {
	active, _, err := registry.LoadActive(p)
	if err != nil {
		return err
	}
	saved, _, err := registry.LoadSaved(p)
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(w)
		return enc.Encode(struct {
			Active    map[string]registry.Session  `json:"active"`
			Groups    map[string]registry.Group    `json:"groups"`
			Bookmarks map[string]registry.Bookmark `json:"bookmarks"`
		}{active.Sessions, saved.Groups, saved.Bookmarks})
	}

	fmt.Fprintln(w, "Active sessions:")
	ids := sortedSessionIDs(active.Sessions)
	if len(ids) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, id := range ids {
		s := active.Sessions[id]
		fmt.Fprintf(w, "  %-8s %-30s %s  (%s)\n", shortID(id), s.Label, s.Cwd, formatAge(now.Sub(s.LastSeen)))
	}

	fmt.Fprintln(w, "\nBookmarks:")
	if len(saved.Bookmarks) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	names := make([]string, 0, len(saved.Bookmarks))
	for n := range saved.Bookmarks {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b := saved.Bookmarks[n]
		fmt.Fprintf(w, "  %-20s %-30s %s\n", n, b.Label, b.Cwd)
	}

	fmt.Fprintln(w, "\nGroups:")
	if len(saved.Groups) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	gnames := make([]string, 0, len(saved.Groups))
	for n := range saved.Groups {
		gnames = append(gnames, n)
	}
	sort.Strings(gnames)
	for _, n := range gnames {
		g := saved.Groups[n]
		fmt.Fprintf(w, "  %-20s %d session(s), saved %s\n", n, len(g.Sessions), formatAge(now.Sub(g.SavedAt)))
	}
	return nil
}

func sortedSessionIDs(sessions map[string]registry.Session) []string {
	ids := make([]string, 0, len(sessions))
	for id := range sessions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return sessions[ids[i]].LastSeen.After(sessions[ids[j]].LastSeen)
	})
	return ids
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// formatAge renders a duration the way "last seen" ages are shown to users:
// coarse buckets, never more precise than the user needs for a picker.
func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
