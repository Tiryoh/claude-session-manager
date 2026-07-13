package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/claudedir"
	"github.com/Tiryoh/claude-session-manager/internal/procscan"
	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// RunList prints the active sessions, saved groups, and bookmarks in p. With
// jsonOut it prints one JSON object instead of the human-readable form.
// firstMessage selects each session's first prompt instead of its most
// recent one, which is read live from its transcript under claudeDir. scan
// (e.g. procscan.Scan) reports which sessions still have a live claude
// process; an active session it doesn't recognize is flagged as not
// running.
func RunList(w io.Writer, p registry.Paths, claudeDir string, jsonOut bool, firstMessage bool, scan func() (procscan.Snapshot, bool), now time.Time) error {
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

	snap, scanOK := scan()

	fmt.Fprintln(w, "Active sessions:")
	ids := sortedSessionIDs(active.Sessions)
	if len(ids) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, id := range ids {
		s := active.Sessions[id]
		fmt.Fprintf(w, "  session: %s\n", id)
		fmt.Fprintf(w, "  dir:     %s\n", s.Cwd)
		fmt.Fprintf(w, "  message: %s\n", sessionMessage(claudeDir, s, firstMessage))
		seen := formatAge(now.Sub(s.LastSeen))
		if scanOK && !snap.IsRunning(id, s.Cwd) {
			seen += "  [not running]"
		}
		fmt.Fprintf(w, "  seen:    %s\n\n", seen)
	}

	fmt.Fprintln(w, "Bookmarks:")
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
		fmt.Fprintf(w, "  %s\n", n)
		fmt.Fprintf(w, "    session: %s\n", b.SessionID)
		fmt.Fprintf(w, "    dir:     %s\n", b.Cwd)
		fmt.Fprintf(w, "    message: %s\n", sessionMessage(claudeDir, b.Session, firstMessage))
		fmt.Fprintf(w, "    saved:   %s\n\n", formatAge(now.Sub(b.SavedAt)))
	}

	fmt.Fprintln(w, "Groups:")
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
		fmt.Fprintf(w, "  %s\n", n)
		fmt.Fprintf(w, "    sessions: %d\n", len(g.Sessions))
		fmt.Fprintf(w, "    saved:    %s\n\n", formatAge(now.Sub(g.SavedAt)))
	}
	return nil
}

// sessionMessage picks which prompt to show for s: the first one (Label) if
// firstMessage is set, otherwise the most recent one, read live from s's
// transcript under claudeDir. It falls back to Label when the transcript
// has no last-prompt entry (yet), e.g. right after SessionStart.
func sessionMessage(claudeDir string, s registry.Session, firstMessage bool) string {
	if !firstMessage {
		if msg, ok := claudedir.LastPrompt(claudeDir, s.Cwd, s.SessionID); ok {
			return displayMessage(truncateMessage(msg))
		}
	}
	return displayMessage(s.Label)
}

// truncateMessage collapses whitespace (including newlines) into single
// spaces and caps the result at 80 runes so list output stays one line.
func truncateMessage(s string) string {
	joined := strings.Join(strings.Fields(s), " ")
	r := []rune(joined)
	if len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return joined
}

// displayMessage returns msg, or a placeholder when no message has been
// recorded yet (e.g. a session with no UserPromptSubmit so far).
func displayMessage(msg string) string {
	if msg == "" {
		return "(no message yet)"
	}
	return msg
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
