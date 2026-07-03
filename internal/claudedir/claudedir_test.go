package claudedir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeProjectDir(t *testing.T) {
	got := EncodeProjectDir("/home/daisuke/ghq/github.com/Tiryoh/claude-session-manager")
	want := "-home-daisuke-ghq-github-com-Tiryoh-claude-session-manager"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranscriptPath(t *testing.T) {
	got := TranscriptPath("/home/u/.claude", "/tmp/proj", "abc-123")
	want := filepath.Join("/home/u/.claude", "projects", "-tmp-proj", "abc-123.jsonl")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranscriptExists(t *testing.T) {
	claudeDir := t.TempDir()
	projDir := filepath.Join(claudeDir, "projects", EncodeProjectDir("/tmp/proj"))
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "abc.jsonl"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !TranscriptExists(claudeDir, "/tmp/proj", "abc") {
		t.Fatal("expected transcript to exist")
	}
	if TranscriptExists(claudeDir, "/tmp/proj", "missing") {
		t.Fatal("expected transcript to not exist")
	}
}

func TestDefaultClaudeDir_UsesEnvOverride(t *testing.T) {
	t.Setenv("CSM_CLAUDE_DIR", "/custom/claude")
	got, err := DefaultClaudeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/claude" {
		t.Fatalf("got %q, want /custom/claude", got)
	}
}
