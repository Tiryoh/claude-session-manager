// Package install manages csm's hook entries inside Claude Code's
// ~/.claude/settings.json, merging them in without disturbing any other
// key the user (or another tool) has configured there.
package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
)

// HookCmd is one command entry inside a HookEntry's "hooks" array, matching
// Claude Code's settings.json schema.
type HookCmd struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// HookEntry is one matcher block for a given hook event.
type HookEntry struct {
	Matcher string    `json:"matcher,omitempty"`
	Hooks   []HookCmd `json:"hooks"`
}

// HooksSection is the value of the top-level "hooks" key in settings.json.
type HooksSection map[string][]HookEntry

// ourCommandRe identifies a HookCmd as one csm installed, regardless of the
// absolute path the binary was installed at.
var ourCommandRe = regexp.MustCompile(`(^|/)csm hook$`)

func isOurs(command string) bool {
	return ourCommandRe.MatchString(command)
}

var managedEvents = []struct {
	name    string
	matcher string
}{
	{"SessionStart", "startup|resume|clear"},
	{"UserPromptSubmit", ""},
	{"SessionEnd", ""},
}

// Install merges csm's hook entries into the settings.json at path,
// preserving every other key byte-for-byte (only "hooks" is re-encoded).
// If path doesn't exist yet, a new settings.json is created. Unless dryRun
// is set and a change is needed, the original is backed up to path+".bak"
// before the new content is written. changed reports whether the merge
// altered anything (so callers can skip the backup/write and say "already
// installed").
func Install(path, exePath string, dryRun bool) ([]byte, bool, error) {
	top, original, existed, err := readSettings(path)
	if err != nil {
		return nil, false, err
	}

	hooks, err := decodeHooks(top)
	if err != nil {
		return nil, false, err
	}
	before, _ := json.Marshal(hooks)
	for _, e := range managedEvents {
		mergeEvent(hooks, e.name, e.matcher, exePath)
	}
	after, _ := json.Marshal(hooks)
	changed := !bytes.Equal(before, after)

	rendered, err := renderSettings(top, hooks)
	if err != nil {
		return nil, false, err
	}
	if dryRun || !changed {
		return rendered, changed, nil
	}
	if existed {
		if err := os.WriteFile(path+".bak", original, 0o644); err != nil {
			return nil, false, err
		}
	}
	if err := os.WriteFile(path, rendered, 0o644); err != nil {
		return nil, false, err
	}
	return rendered, changed, nil
}

// Uninstall removes only csm's hook entries from path, leaving every other
// hook and top-level key untouched. It always backs up before writing when
// a change is made.
func Uninstall(path string) ([]byte, bool, error) {
	top, original, existed, err := readSettings(path)
	if err != nil {
		return nil, false, err
	}
	if !existed {
		return nil, false, nil
	}

	hooks, err := decodeHooks(top)
	if err != nil {
		return nil, false, err
	}
	before, _ := json.Marshal(hooks)
	for _, e := range managedEvents {
		removeOurs(hooks, e.name)
	}
	after, _ := json.Marshal(hooks)
	changed := !bytes.Equal(before, after)
	if !changed {
		return original, false, nil
	}

	rendered, err := renderSettings(top, hooks)
	if err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(path+".bak", original, 0o644); err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(path, rendered, 0o644); err != nil {
		return nil, false, err
	}
	return rendered, true, nil
}

func readSettings(path string) (top map[string]json.RawMessage, original []byte, existed bool, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]json.RawMessage{}, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, err
	}
	top = map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, nil, false, fmt.Errorf("parse %s: %w", path, err)
	}
	return top, data, true, nil
}

func decodeHooks(top map[string]json.RawMessage) (HooksSection, error) {
	hooks := HooksSection{}
	if raw, ok := top["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooks); err != nil {
			return nil, fmt.Errorf("parse hooks: %w", err)
		}
	}
	return hooks, nil
}

func renderSettings(top map[string]json.RawMessage, hooks HooksSection) ([]byte, error) {
	if len(hooks) == 0 {
		delete(top, "hooks")
	} else {
		raw, err := json.MarshalIndent(hooks, "", "  ")
		if err != nil {
			return nil, err
		}
		top["hooks"] = raw
	}
	return json.MarshalIndent(top, "", "  ")
}

// mergeEvent removes any pre-existing csm entries for event, then adds our
// command to the HookEntry whose matcher already equals matcher, creating
// one if none matches.
func mergeEvent(hooks HooksSection, event, matcher, exePath string) {
	kept := removeOursFromSlice(hooks[event])
	ourCmd := HookCmd{Type: "command", Command: exePath + " hook", Timeout: 5}

	for i := range kept {
		if kept[i].Matcher == matcher {
			kept[i].Hooks = append(kept[i].Hooks, ourCmd)
			hooks[event] = kept
			return
		}
	}
	kept = append(kept, HookEntry{Matcher: matcher, Hooks: []HookCmd{ourCmd}})
	hooks[event] = kept
}

// removeOurs strips csm's entries for event, deleting the event key
// entirely if nothing else is left.
func removeOurs(hooks HooksSection, event string) {
	kept := removeOursFromSlice(hooks[event])
	if len(kept) == 0 {
		delete(hooks, event)
	} else {
		hooks[event] = kept
	}
}

func removeOursFromSlice(entries []HookEntry) []HookEntry {
	var kept []HookEntry
	for _, e := range entries {
		var remaining []HookCmd
		for _, c := range e.Hooks {
			if !isOurs(c.Command) {
				remaining = append(remaining, c)
			}
		}
		if len(remaining) > 0 {
			e.Hooks = remaining
			kept = append(kept, e)
		}
	}
	return kept
}
