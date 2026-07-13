package claudedir

import (
	"os"
	"path/filepath"
	"strings"
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

func writeTranscript(t *testing.T, claudeDir, cwd, sessionID string, lines []string) {
	t.Helper()
	path := TranscriptPath(claudeDir, cwd, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLastPrompt_ReturnsMostRecentEntry(t *testing.T) {
	claudeDir := t.TempDir()
	writeTranscript(t, claudeDir, "/tmp/proj", "abc", []string{
		`{"type":"user","message":{"content":"noise, not a last-prompt entry"}}`,
		`{"type":"last-prompt","lastPrompt":"first prompt"}`,
		`{"type":"assistant","message":{"content":"some reply"}}`,
		`{"type":"last-prompt","lastPrompt":"second prompt"}`,
	})

	got, ok := LastPrompt(claudeDir, "/tmp/proj", "abc")
	if !ok {
		t.Fatal("LastPrompt() ok = false, want true")
	}
	if got != "second prompt" {
		t.Fatalf("LastPrompt() = %q, want %q", got, "second prompt")
	}
}

func TestLastPrompt_NoEntryFound(t *testing.T) {
	claudeDir := t.TempDir()
	writeTranscript(t, claudeDir, "/tmp/proj", "abc", []string{
		`{"type":"user","message":{"content":"hi"}}`,
	})

	if _, ok := LastPrompt(claudeDir, "/tmp/proj", "abc"); ok {
		t.Fatal("LastPrompt() ok = true, want false when no last-prompt entry exists")
	}
}

func TestLastPrompt_MissingTranscript(t *testing.T) {
	claudeDir := t.TempDir()
	if _, ok := LastPrompt(claudeDir, "/tmp/proj", "missing"); ok {
		t.Fatal("LastPrompt() ok = true, want false for a missing transcript")
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
