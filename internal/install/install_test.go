package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const realisticSettings = `{
  "cleanupPeriodDays": 365,
  "model": "sonnet",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "echo pretooluse", "timeout": 5 }
        ]
      }
    ]
  },
  "statusLine": { "type": "command", "command": "npx -y ccstatusline@latest", "padding": 0 },
  "enabledPlugins": { "superpowers@claude-plugins-official": true }
}`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInstall_PreservesUnrelatedKeys(t *testing.T) {
	path := writeFixture(t, realisticSettings)

	rendered, changed, err := Install(path, "/home/u/go/bin/csm", false)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if !changed {
		t.Fatalf("changed = false on first install, want true")
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(rendered, &got); err != nil {
		t.Fatalf("rendered output is not valid JSON: %v", err)
	}
	if string(got["model"]) != `"sonnet"` {
		t.Fatalf("model key was altered: %s", got["model"])
	}
	if !strings.Contains(string(got["enabledPlugins"]), "superpowers@claude-plugins-official") {
		t.Fatalf("enabledPlugins key was altered: %s", got["enabledPlugins"])
	}
	if !strings.Contains(string(got["hooks"]), "echo pretooluse") {
		t.Fatalf("existing PreToolUse hook was dropped: %s", got["hooks"])
	}

	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup file, stat error = %v", err)
	}
}

func TestInstall_AddsSessionStartUserPromptSubmitSessionEnd(t *testing.T) {
	path := writeFixture(t, realisticSettings)
	rendered, _, err := Install(path, "/home/u/go/bin/csm", false)
	if err != nil {
		t.Fatal(err)
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(rendered, &top); err != nil {
		t.Fatal(err)
	}
	var hooks HooksSection
	if err := json.Unmarshal(top["hooks"], &hooks); err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{"SessionStart", "UserPromptSubmit", "SessionEnd"} {
		entries, ok := hooks[event]
		if !ok || len(entries) == 0 {
			t.Fatalf("missing %s entries: %+v", event, hooks)
		}
		found := false
		for _, e := range entries {
			for _, c := range e.Hooks {
				if c.Command == "/home/u/go/bin/csm hook" {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("%s does not contain our hook command: %+v", event, entries)
		}
	}
	ss := hooks["SessionStart"][0]
	if ss.Matcher != "startup|resume|clear" {
		t.Fatalf("SessionStart matcher = %q, want startup|resume|clear", ss.Matcher)
	}
}

func TestInstall_IsIdempotent(t *testing.T) {
	path := writeFixture(t, realisticSettings)
	first, _, err := Install(path, "/home/u/go/bin/csm", false)
	if err != nil {
		t.Fatal(err)
	}
	second, changed, err := Install(path, "/home/u/go/bin/csm", false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("second Install() reported changed=true, want false (idempotent)")
	}
	if string(first) != string(second) {
		t.Fatalf("second install produced different output:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestUninstall_RemovesOnlyOurEntries(t *testing.T) {
	path := writeFixture(t, realisticSettings)
	if _, _, err := Install(path, "/home/u/go/bin/csm", false); err != nil {
		t.Fatal(err)
	}

	rendered, changed, err := Uninstall(path)
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if !changed {
		t.Fatalf("changed = false, want true")
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(rendered, &top); err != nil {
		t.Fatal(err)
	}
	var hooks HooksSection
	if err := json.Unmarshal(top["hooks"], &hooks); err != nil {
		t.Fatal(err)
	}
	if _, ok := hooks["SessionStart"]; ok {
		t.Fatalf("SessionStart should be fully removed (csm was its only entry): %+v", hooks)
	}
	if len(hooks["PreToolUse"]) != 1 || !strings.Contains(hooks["PreToolUse"][0].Hooks[0].Command, "echo pretooluse") {
		t.Fatalf("PreToolUse entry was disturbed: %+v", hooks["PreToolUse"])
	}
}

func TestInstall_CreatesFileWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	_, changed, err := Install(path, "/home/u/go/bin/csm", false)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if !changed {
		t.Fatalf("changed = false, want true")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Fatalf(".bak should not be created when there was no original file")
	}
}

func TestInstall_DryRunDoesNotWrite(t *testing.T) {
	path := writeFixture(t, realisticSettings)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, changed, err := Install(path, "/home/u/go/bin/csm", true)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatalf("changed = false, want true even in dry-run (reflects what would happen)")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("dry-run modified the file on disk")
	}
}
