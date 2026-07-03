// Package registry stores and retrieves csm's session data: the active
// sessions currently tracked via Claude Code hooks, and the groups/
// bookmarks the user has explicitly saved.
package registry

import (
	"os"
	"path/filepath"
)

// Paths resolves the on-disk locations csm uses for its registry files.
type Paths struct {
	ConfigDir string
}

// DefaultPaths resolves the config directory: $CSM_CONFIG_DIR if set,
// otherwise os.UserConfigDir()/claude-session-manager.
func DefaultPaths() (Paths, error) {
	if dir := os.Getenv("CSM_CONFIG_DIR"); dir != "" {
		return Paths{ConfigDir: dir}, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	return Paths{ConfigDir: filepath.Join(base, "claude-session-manager")}, nil
}

func (p Paths) ActiveFile() string  { return filepath.Join(p.ConfigDir, "active.json") }
func (p Paths) SavedFile() string   { return filepath.Join(p.ConfigDir, "saved.json") }
func (p Paths) HookLogFile() string { return filepath.Join(p.ConfigDir, "hook.log") }
