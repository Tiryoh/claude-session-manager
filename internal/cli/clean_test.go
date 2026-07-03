package cli

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

func TestRunClean_RemovesEntriesWithMissingTranscript(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	cwdWithTranscript := t.TempDir()
	cwdWithout := t.TempDir()
	now := time.Now()

	seedTranscript(t, claudeDir, cwdWithTranscript, "has-transcript")
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["has-transcript"] = registry.Session{SessionID: "has-transcript", Cwd: cwdWithTranscript, LastSeen: now}
		r.Sessions["no-transcript"] = registry.Session{SessionID: "no-transcript", Cwd: cwdWithout, LastSeen: now}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunClean(&out, p, claudeDir, 720*time.Hour, false, now); err != nil {
		t.Fatalf("RunClean() error = %v", err)
	}

	reg, _, err := registry.LoadActive(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Sessions["has-transcript"]; !ok {
		t.Fatal("session with an existing transcript should survive")
	}
	if _, ok := reg.Sessions["no-transcript"]; ok {
		t.Fatal("session with a missing transcript should be pruned")
	}
}

func TestRunClean_RemovesEntriesOlderThanThreshold(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	cwd := t.TempDir()
	seedTranscript(t, claudeDir, cwd, "old-session")
	seedTranscript(t, claudeDir, cwd, "recent-session")
	now := time.Now()

	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["old-session"] = registry.Session{SessionID: "old-session", Cwd: cwd, LastSeen: now.Add(-31 * 24 * time.Hour)}
		r.Sessions["recent-session"] = registry.Session{SessionID: "recent-session", Cwd: cwd, LastSeen: now.Add(-1 * time.Hour)}
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := RunClean(&bytes.Buffer{}, p, claudeDir, 30*24*time.Hour, false, now); err != nil {
		t.Fatal(err)
	}

	reg, _, err := registry.LoadActive(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Sessions["old-session"]; ok {
		t.Fatal("session older than the threshold should be pruned")
	}
	if _, ok := reg.Sessions["recent-session"]; !ok {
		t.Fatal("recent session (power-loss survivor) must NOT be pruned just for being idle")
	}
}

func TestRunClean_DryRunChangesNothing(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	now := time.Now()
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		r.Sessions["gone"] = registry.Session{SessionID: "gone", Cwd: filepath.Join(t.TempDir(), "missing"), LastSeen: now}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunClean(&out, p, claudeDir, 720*time.Hour, true, now); err != nil {
		t.Fatal(err)
	}

	reg, _, err := registry.LoadActive(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Sessions["gone"]; !ok {
		t.Fatal("dry-run must not remove anything")
	}
	if out.Len() == 0 {
		t.Fatal("dry-run should still report what it would remove")
	}
}
