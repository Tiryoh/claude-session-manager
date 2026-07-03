package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Session is a single Claude Code session tracked by csm, either currently
// active or captured as a snapshot inside a saved Group/Bookmark.
type Session struct {
	SessionID string    `json:"session_id"`
	Cwd       string    `json:"cwd"`
	Label     string    `json:"label"`
	StartedAt time.Time `json:"started_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// ActiveRegistry is the live set of sessions csm has seen SessionStart/
// UserPromptSubmit events for and no matching SessionEnd yet.
type ActiveRegistry struct {
	Version  int                `json:"version"`
	Sessions map[string]Session `json:"sessions"`
}

// Group is a named, timestamped snapshot of several sessions saved together.
type Group struct {
	SavedAt  time.Time `json:"saved_at"`
	Sessions []Session `json:"sessions"`
}

// Bookmark is a named snapshot of a single session.
type Bookmark struct {
	Session
	SavedAt time.Time `json:"saved_at"`
}

// SavedRegistry holds all named groups and bookmarks.
type SavedRegistry struct {
	Version   int                 `json:"version"`
	Groups    map[string]Group    `json:"groups"`
	Bookmarks map[string]Bookmark `json:"bookmarks"`
}

func newActiveRegistry() *ActiveRegistry {
	return &ActiveRegistry{Version: 1, Sessions: map[string]Session{}}
}

func newSavedRegistry() *SavedRegistry {
	return &SavedRegistry{Version: 1, Groups: map[string]Group{}, Bookmarks: map[string]Bookmark{}}
}

// LoadActive reads active.json, recovering from a corrupt file by moving it
// aside and returning an empty registry instead of failing.
func LoadActive(p Paths) (*ActiveRegistry, bool, error) {
	reg := newActiveRegistry()
	recovered, err := loadJSON(p.ActiveFile(), reg)
	if err != nil {
		return nil, false, err
	}
	if reg.Sessions == nil {
		reg.Sessions = map[string]Session{}
	}
	return reg, recovered, nil
}

// LoadSaved reads saved.json with the same recovery semantics as LoadActive.
func LoadSaved(p Paths) (*SavedRegistry, bool, error) {
	reg := newSavedRegistry()
	recovered, err := loadJSON(p.SavedFile(), reg)
	if err != nil {
		return nil, false, err
	}
	if reg.Groups == nil {
		reg.Groups = map[string]Group{}
	}
	if reg.Bookmarks == nil {
		reg.Bookmarks = map[string]Bookmark{}
	}
	return reg, recovered, nil
}

// loadJSON unmarshals path into v. A missing or empty file is treated as "no
// data yet" (v keeps its zero value, no error). A file that fails to parse is
// renamed to "<path>.corrupt-<unix-ts>" and treated the same as missing.
func loadJSON(path string, v any) (recovered bool, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		backup := path + ".corrupt-" + strconv.FormatInt(time.Now().Unix(), 10)
		_ = os.Rename(path, backup)
		return true, nil
	}
	return false, nil
}

// MutateActive loads active.json under an exclusive lock, applies fn, and
// atomically writes the result back before releasing the lock.
func MutateActive(p Paths, fn func(*ActiveRegistry)) error {
	return withLock(p.ConfigDir, "active", func() error {
		reg, _, err := LoadActive(p)
		if err != nil {
			return err
		}
		fn(reg)
		return writeAtomic(p.ActiveFile(), reg)
	})
}

// MutateSaved loads saved.json under an exclusive lock, applies fn, and
// atomically writes the result back before releasing the lock.
func MutateSaved(p Paths, fn func(*SavedRegistry)) error {
	return withLock(p.ConfigDir, "saved", func() error {
		reg, _, err := LoadSaved(p)
		if err != nil {
			return err
		}
		fn(reg)
		return writeAtomic(p.SavedFile(), reg)
	})
}

// withLock creates dir if needed, takes an exclusive flock on
// "<name>.lock", runs fn, then releases the lock.
func withLock(dir, name string, fn func() error) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	lockPath := filepath.Join(dir, name+".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

// writeAtomic marshals v as indented JSON and writes it to path via a
// temp-file-then-rename so readers never observe a partial write.
func writeAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
