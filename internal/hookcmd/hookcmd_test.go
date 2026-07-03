package hookcmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

func testPaths(t *testing.T) registry.Paths {
	t.Helper()
	return registry.Paths{ConfigDir: t.TempDir()}
}

func TestRun_SessionStart_CreatesEntry(t *testing.T) {
	p := testPaths(t)
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	body := `{"session_id":"s1","cwd":"/tmp/proj","hook_event_name":"SessionStart"}`

	Run(strings.NewReader(body), p, now)

	reg, _, err := registry.LoadActive(p)
	if err != nil {
		t.Fatalf("LoadActive() error = %v", err)
	}
	got, ok := reg.Sessions["s1"]
	if !ok {
		t.Fatalf("session s1 not created")
	}
	if got.Cwd != "/tmp/proj" || !got.StartedAt.Equal(now) || !got.LastSeen.Equal(now) {
		t.Fatalf("got %+v", got)
	}
}

func TestRun_UserPromptSubmit_SetsLabelOnce(t *testing.T) {
	p := testPaths(t)
	t1 := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Minute)

	Run(strings.NewReader(`{"session_id":"s1","cwd":"/x","hook_event_name":"SessionStart"}`), p, t1)
	Run(strings.NewReader(`{"session_id":"s1","cwd":"/x","hook_event_name":"UserPromptSubmit","prompt":"fix the flaky test"}`), p, t2)
	Run(strings.NewReader(`{"session_id":"s1","cwd":"/x","hook_event_name":"UserPromptSubmit","prompt":"second prompt"}`), p, t2.Add(time.Minute))

	reg, _, err := registry.LoadActive(p)
	if err != nil {
		t.Fatal(err)
	}
	got := reg.Sessions["s1"]
	if got.Label != "fix the flaky test" {
		t.Fatalf("Label = %q, want first prompt to stick", got.Label)
	}
	if !got.LastSeen.Equal(t2.Add(time.Minute)) {
		t.Fatalf("LastSeen not advanced on later prompts")
	}
}

func TestRun_SessionEnd_RemovesEntry(t *testing.T) {
	p := testPaths(t)
	now := time.Now()
	Run(strings.NewReader(`{"session_id":"s1","cwd":"/x","hook_event_name":"SessionStart"}`), p, now)
	Run(strings.NewReader(`{"session_id":"s1","cwd":"/x","hook_event_name":"SessionEnd"}`), p, now)

	reg, _, err := registry.LoadActive(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Sessions["s1"]; ok {
		t.Fatalf("session s1 still present after SessionEnd")
	}
}

func TestRun_GarbageInput_NeverPanicsOrErrorsCaller(t *testing.T) {
	p := testPaths(t)
	// No panic is the assertion; Run has no error return by design.
	Run(strings.NewReader("not json at all"), p, time.Now())
	Run(strings.NewReader(""), p, time.Now())
	Run(strings.NewReader(`{"hook_event_name":"SessionStart"}`), p, time.Now()) // missing session_id

	if _, err := os.Stat(p.HookLogFile()); err != nil {
		t.Fatalf("expected hook.log to record the errors, stat error = %v", err)
	}
}

func TestRun_UnknownEvent_RefreshesLastSeenIfPresent(t *testing.T) {
	p := testPaths(t)
	t1 := time.Now()
	t2 := t1.Add(time.Hour)
	Run(strings.NewReader(`{"session_id":"s1","cwd":"/x","hook_event_name":"SessionStart"}`), p, t1)
	Run(strings.NewReader(`{"session_id":"s1","cwd":"/x","hook_event_name":"SomeFutureEvent"}`), p, t2)

	reg, _, err := registry.LoadActive(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reg.Sessions["s1"].LastSeen.Equal(t2) {
		t.Fatalf("LastSeen not refreshed by unknown event")
	}
}

func TestTruncateLabel(t *testing.T) {
	long := strings.Repeat("word ", 30) // 150 chars incl separators
	got := truncateLabel(long)
	if r := []rune(got); len(r) > 81 { // 80 + ellipsis
		t.Fatalf("label too long: %d runes", len(r))
	}
	if strings.Contains(truncateLabel("line one\nline two"), "\n") {
		t.Fatalf("label must not contain newlines")
	}
}
