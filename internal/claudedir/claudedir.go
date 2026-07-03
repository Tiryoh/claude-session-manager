// Package claudedir locates Claude Code's own data directory and the
// transcript files it writes there, so csm can tell whether a saved session
// is still restorable.
package claudedir

import (
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
