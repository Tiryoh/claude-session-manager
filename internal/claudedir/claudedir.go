// Package claudedir locates Claude Code's own data directory and the
// transcript files it writes there, so csm can tell whether a saved session
// is still restorable.
package claudedir

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

// EncodeProjectDir reproduces Claude Code's project-directory encoding:
// every byte that isn't a letter or digit becomes '-'. This mirrors the
// naming under ~/.claude/projects/ but is not an officially documented
// format, so callers must treat a lookup miss as "unknown", never fatal.
func EncodeProjectDir(cwd string) string {
	out := make([]byte, len(cwd))
	for i := 0; i < len(cwd); i++ {
		c := cwd[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out[i] = c
		default:
			out[i] = '-'
		}
	}
	return string(out)
}

// TranscriptPath returns where Claude Code would store the transcript for
// sessionID started in cwd, under claudeDir (normally ~/.claude).
func TranscriptPath(claudeDir, cwd, sessionID string) string {
	return filepath.Join(claudeDir, "projects", EncodeProjectDir(cwd), sessionID+".jsonl")
}

// TranscriptExists reports whether that transcript file is present. A false
// result means "likely expired / already cleaned up", used only for display
// warnings -- never treated as a hard failure.
func TranscriptExists(claudeDir, cwd, sessionID string) bool {
	_, err := os.Stat(TranscriptPath(claudeDir, cwd, sessionID))
	return err == nil
}

// lastPromptTailBytes bounds how much of a transcript LastPrompt reads: it
// only ever needs the final "last-prompt" entry, which Claude Code appends
// once per user turn, so this comfortably covers even a very chatty final
// turn without loading a whole (potentially multi-GB) transcript into memory.
const lastPromptTailBytes = 64 * 1024

// LastPrompt returns the most recent user prompt recorded for sessionID, as
// tracked in Claude Code's own transcript via "last-prompt" entries -- this
// lets csm show it without having to parse tool-result/thinking/text content
// shapes out of the raw (and undocumented) transcript format itself. ok is
// false if the transcript is missing or has no last-prompt entry in the part
// read.
func LastPrompt(claudeDir, cwd, sessionID string) (string, bool) {
	f, err := os.Open(TranscriptPath(claudeDir, cwd, sessionID))
	if err != nil {
		return "", false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", false
	}
	start := int64(0)
	if size := info.Size(); size > lastPromptTailBytes {
		start = size - lastPromptTailBytes
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", false
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return "", false
	}

	prompt, ok := "", false
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		var entry struct {
			Type       string `json:"type"`
			LastPrompt string `json:"lastPrompt"`
		}
		if json.Unmarshal(line, &entry) != nil || entry.Type != "last-prompt" || entry.LastPrompt == "" {
			continue
		}
		prompt, ok = entry.LastPrompt, true
	}
	return prompt, ok
}

// DefaultClaudeDir resolves Claude Code's data directory: $CSM_CLAUDE_DIR if
// set, otherwise ~/.claude.
func DefaultClaudeDir() (string, error) {
	if dir := os.Getenv("CSM_CLAUDE_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}
