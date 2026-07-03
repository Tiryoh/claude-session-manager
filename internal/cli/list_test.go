package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

func TestRunList_TextOutputShowsActiveAndSaved(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
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
	if err := RunList(&out, p, false, now); err != nil {
		t.Fatalf("RunList() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"s1"[:2], "fix bug", "/proj", "robot-fw", "sprint-23"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestRunList_JSONOutput(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	now := time.Now()
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["s1"] = registry.Session{SessionID: "s1", Cwd: "/proj"}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunList(&out, p, true, now); err != nil {
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
