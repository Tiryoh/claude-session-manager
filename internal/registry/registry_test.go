package registry

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	return Paths{ConfigDir: t.TempDir()}
}

func TestLoadActive_MissingFileReturnsEmpty(t *testing.T) {
	p := testPaths(t)
	reg, recovered, err := LoadActive(p)
	if err != nil {
		t.Fatalf("LoadActive() error = %v", err)
	}
	if recovered {
		t.Fatalf("recovered = true for missing file, want false")
	}
	if reg.Version != 1 || len(reg.Sessions) != 0 {
		t.Fatalf("got %+v, want empty v1 registry", reg)
	}
}

func TestMutateActive_UpsertAndPersist(t *testing.T) {
	p := testPaths(t)
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	err := MutateActive(p, func(r *ActiveRegistry) {
		r.Sessions["abc"] = Session{SessionID: "abc", Cwd: "/tmp/x", StartedAt: now, LastSeen: now}
	})
	if err != nil {
		t.Fatalf("MutateActive() error = %v", err)
	}

	reg, _, err := LoadActive(p)
	if err != nil {
		t.Fatalf("LoadActive() error = %v", err)
	}
	got, ok := reg.Sessions["abc"]
	if !ok {
		t.Fatalf("session %q not persisted", "abc")
	}
	if got.Cwd != "/tmp/x" || !got.LastSeen.Equal(now) {
		t.Fatalf("got %+v, want Cwd=/tmp/x LastSeen=%v", got, now)
	}
}

func TestLoadActive_CorruptFileRecovers(t *testing.T) {
	p := testPaths(t)
	if err := os.MkdirAll(p.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.ActiveFile(), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg, recovered, err := LoadActive(p)
	if err != nil {
		t.Fatalf("LoadActive() error = %v, want recovery not error", err)
	}
	if !recovered {
		t.Fatalf("recovered = false, want true")
	}
	if len(reg.Sessions) != 0 {
		t.Fatalf("got %d sessions, want 0 after recovery", len(reg.Sessions))
	}

	matches, _ := filepath.Glob(filepath.Join(p.ConfigDir, "active.json.corrupt-*"))
	if len(matches) != 1 {
		t.Fatalf("found %d corrupt-backup files, want 1", len(matches))
	}
}

func TestMutateSaved_GroupsAndBookmarks(t *testing.T) {
	p := testPaths(t)
	now := time.Now()

	err := MutateSaved(p, func(r *SavedRegistry) {
		r.Groups["sprint-23"] = Group{SavedAt: now, Sessions: []Session{{SessionID: "s1", Cwd: "/a"}}}
		r.Bookmarks["robot-fw"] = Bookmark{Session: Session{SessionID: "b1", Cwd: "/b"}, SavedAt: now}
	})
	if err != nil {
		t.Fatalf("MutateSaved() error = %v", err)
	}

	reg, _, err := LoadSaved(p)
	if err != nil {
		t.Fatalf("LoadSaved() error = %v", err)
	}
	if len(reg.Groups["sprint-23"].Sessions) != 1 {
		t.Fatalf("group not persisted: %+v", reg.Groups)
	}
	if reg.Bookmarks["robot-fw"].Cwd != "/b" {
		t.Fatalf("bookmark not persisted: %+v", reg.Bookmarks)
	}
}

func TestMutateActive_ConcurrentWritesAllSurvive(t *testing.T) {
	p := testPaths(t)
	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			id := filepath.Base(t.TempDir()) // cheap unique-ish string per goroutine
			err := MutateActive(p, func(r *ActiveRegistry) {
				r.Sessions[id+string(rune('a'+i))] = Session{SessionID: id, Cwd: "/x"}
			})
			if err != nil {
				t.Errorf("MutateActive() error = %v", err)
			}
		}(i)
	}
	wg.Wait()

	reg, _, err := LoadActive(p)
	if err != nil {
		t.Fatalf("LoadActive() error = %v", err)
	}
	if len(reg.Sessions) != n {
		t.Fatalf("got %d sessions, want %d (a lost write means the lock failed)", len(reg.Sessions), n)
	}
}
