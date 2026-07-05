package cli

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

func seedTranscript(t *testing.T, claudeDir, cwd, sessionID string) {
	t.Helper()
	dir := filepath.Join(claudeDir, "projects", encodeForTest(cwd))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sessionID+".jsonl"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// encodeForTest mirrors claudedir.EncodeProjectDir without importing it, to
// keep this test file focused on RunOpen's own contract. Task 4 owns the
// real encoding and its own tests.
func encodeForTest(cwd string) string {
	out := []byte(cwd)
	for i, c := range out {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			out[i] = '-'
		}
	}
	return string(out)
}

func TestRunOpen_RejectsWithoutTTY(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	var out bytes.Buffer
	err := RunOpen(&out, bufio.NewReader(strings.NewReader("")), p, t.TempDir(), "", false, false, nil, time.Now())
	if err == nil {
		t.Fatal("expected an error when not running on a TTY")
	}
}

func TestRunOpen_BookmarkExecsInSavedDir(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	cwd := t.TempDir()
	seedTranscript(t, claudeDir, cwd, "sess-1")

	err := registry.MutateSaved(p, func(r *registry.SavedRegistry) {
		r.Bookmarks["robot-fw"] = registry.Bookmark{Session: registry.Session{SessionID: "sess-1", Cwd: cwd, Label: "fix bug"}}
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var gotDir string
	var gotArgv []string
	exec := func(dir string, argv []string) error {
		gotDir, gotArgv = dir, argv
		return nil
	}

	r := bufio.NewReader(strings.NewReader("1\n")) // picker: only one bookmark, choose it
	err = RunOpen(&out, r, p, claudeDir, "", false, true, exec, time.Now())
	if err != nil {
		t.Fatalf("RunOpen() error = %v", err)
	}
	if gotDir != cwd {
		t.Fatalf("exec dir = %q, want %q", gotDir, cwd)
	}
	if len(gotArgv) != 3 || gotArgv[0] != "claude" || gotArgv[1] != "--resume" || gotArgv[2] != "sess-1" {
		t.Fatalf("exec argv = %v", gotArgv)
	}
}

func TestRunOpen_ForkAddsFlag(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	cwd := t.TempDir()
	seedTranscript(t, claudeDir, cwd, "sess-1")
	if err := registry.MutateSaved(p, func(r *registry.SavedRegistry) {
		r.Bookmarks["b"] = registry.Bookmark{Session: registry.Session{SessionID: "sess-1", Cwd: cwd}}
	}); err != nil {
		t.Fatal(err)
	}

	var gotArgv []string
	exec := func(dir string, argv []string) error { gotArgv = argv; return nil }
	var out bytes.Buffer
	err := RunOpen(&out, bufio.NewReader(strings.NewReader("1\n")), p, claudeDir, "", true, true, exec, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range gotArgv {
		if a == "--fork-session" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected --fork-session in argv, got %v", gotArgv)
	}
}

func TestRunOpen_DirectNameSkipsPicker(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	cwd := t.TempDir()
	seedTranscript(t, claudeDir, cwd, "sess-1")
	if err := registry.MutateSaved(p, func(r *registry.SavedRegistry) {
		r.Bookmarks["robot-fw"] = registry.Bookmark{Session: registry.Session{SessionID: "sess-1", Cwd: cwd}}
	}); err != nil {
		t.Fatal(err)
	}

	var gotDir string
	exec := func(dir string, argv []string) error { gotDir = dir; return nil }
	// empty reader: must not need input because the name is unambiguous
	err := RunOpen(&bytes.Buffer{}, bufio.NewReader(strings.NewReader("")), p, claudeDir, "robot-fw", false, true, exec, time.Now())
	if err != nil {
		t.Fatalf("RunOpen() error = %v", err)
	}
	if gotDir != cwd {
		t.Fatalf("gotDir = %q, want %q", gotDir, cwd)
	}
}

func TestRunOpen_MissingCwdRefuses(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	claudeDir := t.TempDir()
	if err := registry.MutateSaved(p, func(r *registry.SavedRegistry) {
		r.Bookmarks["gone"] = registry.Bookmark{Session: registry.Session{SessionID: "sess-1", Cwd: "/no/such/dir"}}
	}); err != nil {
		t.Fatal(err)
	}
	exec := func(dir string, argv []string) error { t.Fatal("exec should not be called"); return nil }
	err := RunOpen(&bytes.Buffer{}, bufio.NewReader(strings.NewReader("")), p, claudeDir, "gone", false, true, exec, time.Now())
	if err == nil {
		t.Fatal("expected an error when the saved cwd no longer exists")
	}
}
