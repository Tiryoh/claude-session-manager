package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/claudedir"
	"github.com/Tiryoh/claude-session-manager/internal/procscan"
	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// noScan simulates a host where the process table couldn't be scanned, so
// RunList never flags an active session as not running.
func noScan() (procscan.Snapshot, bool) { return procscan.Snapshot{}, false }

// writeTranscript creates a minimal transcript file at the path Claude Code
// would use for cwd/sessionID under claudeDir, containing one "last-prompt"
// entry with the given text (or none, if lastPrompt is "").
func writeTranscript(t *testing.T, claudeDir, cwd, sessionID, lastPrompt string) {
	t.Helper()
	path := claudedir.TranscriptPath(claudeDir, cwd, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"system","subtype":"noise"}`
	if lastPrompt != "" {
		entry := struct {
			Type       string `json:"type"`
			LastPrompt string `json:"lastPrompt"`
		}{"last-prompt", lastPrompt}
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		line = string(data)
	}
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunList_TextOutputShowsActiveAndSaved(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["s1"] = registry.Session{SessionID: "s1", Cwd: "/proj", Label: "fix bug", LastSeen: now.Add(-2 * time.Minute)}
	})
	if err != nil {
		t.Fatal(err)
	}
	err = registry.MutateSaved(p, func(r *registry.SavedRegistry) {
		r.Bookmarks["robot-fw"] = registry.Bookmark{Session: registry.Session{SessionID: "b1", Cwd: "/b"}, SavedAt: now}
		r.Groups["sprint-23"] = registry.Group{SavedAt: now, Sessions: []registry.Session{{SessionID: "g1", Cwd: "/g"}}}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunList(&out, p, claudeDir, false, false, noScan, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"s1", "fix bug", "/proj", "robot-fw", "sprint-23"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestRunList_ShowsFullSessionID(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	now := time.Now()
	id := "550e8400-e29b-41d4-a716-446655440000"
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions[id] = registry.Session{SessionID: id, Cwd: "/proj", LastSeen: now}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunList(&out, p, claudeDir, false, false, noScan, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if !strings.Contains(out.String(), id) {
		t.Fatalf("output missing full session id %q:\n%s", id, out.String())
	}
}

func TestRunList_MessageSelection(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	now := time.Now()
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["s1"] = registry.Session{SessionID: "s1", Cwd: "/proj", Label: "first prompt", LastSeen: now}
	})
	if err != nil {
		t.Fatal(err)
	}
	writeTranscript(t, claudeDir, "/proj", "s1", "latest prompt")

	var out bytes.Buffer
	if err := RunList(&out, p, claudeDir, false, false, noScan, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if !strings.Contains(out.String(), "latest prompt") {
		t.Fatalf("default output should show the last message from the transcript:\n%s", out.String())
	}

	out.Reset()
	if err := RunList(&out, p, claudeDir, false, true, noScan, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if !strings.Contains(out.String(), "first prompt") {
		t.Fatalf("--first-message output should show the first message:\n%s", out.String())
	}
}

func TestRunList_MessageFallsBackToLabelWithoutTranscript(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir() // no transcript written
	now := time.Now()
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["s1"] = registry.Session{SessionID: "s1", Cwd: "/proj", Label: "first prompt", LastSeen: now}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunList(&out, p, claudeDir, false, false, noScan, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if !strings.Contains(out.String(), "first prompt") {
		t.Fatalf("expected fallback to Label when no transcript exists:\n%s", out.String())
	}
}

func TestRunList_FlagsSessionsNotRunning(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	now := time.Now()
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["s1"] = registry.Session{SessionID: "s1", Cwd: "/proj", LastSeen: now.Add(-time.Hour)}
	})
	if err != nil {
		t.Fatal(err)
	}

	nothingRunning := func() (procscan.Snapshot, bool) { return procscan.Snapshot{}, true }
	var out bytes.Buffer
	if err := RunList(&out, p, claudeDir, false, false, nothingRunning, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if !strings.Contains(out.String(), "not running") {
		t.Fatalf("expected a not-running marker when the process scan finds nothing:\n%s", out.String())
	}

	sessionRunning := func() (procscan.Snapshot, bool) {
		return procscan.Snapshot{SessionIDs: map[string]bool{"s1": true}}, true
	}
	out.Reset()
	if err := RunList(&out, p, claudeDir, false, false, sessionRunning, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if strings.Contains(out.String(), "not running") {
		t.Fatalf("did not expect a not-running marker when the scan matches the session id:\n%s", out.String())
	}

	out.Reset()
	if err := RunList(&out, p, claudeDir, false, false, noScan, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	if strings.Contains(out.String(), "not running") {
		t.Fatalf("did not expect a not-running marker when the scan itself failed:\n%s", out.String())
	}
}

func TestRunList_JSONOutput(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	now := time.Now()
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["s1"] = registry.Session{SessionID: "s1", Cwd: "/proj"}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunList(&out, p, claudeDir, true, false, noScan, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	var decoded struct {
		Active map[string]registry.Session `json:"active"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if decoded.Active["s1"].Cwd != "/proj" {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestFormatAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{3 * time.Hour, "3h ago"},
		{50 * time.Hour, "2d ago"},
	}
	for _, c := range cases {
		if got := formatAge(c.d); got != c.want {
			t.Errorf("formatAge(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
