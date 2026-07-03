package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

func seedActive(t *testing.T, p registry.Paths, ids ...string) {
	t.Helper()
	err := registry.MutateActive(p, func(r *registry.ActiveRegistry) {
		for _, id := range ids {
			r.Sessions[id] = registry.Session{SessionID: id, Cwd: "/proj/" + id, Label: "label-" + id}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunSave_AllFlagSavesEverythingNoPrompt(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	seedActive(t, p, "s1", "s2")
	now := time.Now()

	var out bytes.Buffer
	err := RunSave(&out, bufio.NewReader(strings.NewReader("")), p, "sprint-23", true, false, now)
	if err != nil {
		t.Fatalf("RunSave() error = %v", err)
	}

	saved, _, err := registry.LoadSaved(p)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := saved.Groups["sprint-23"]
	if !ok || len(g.Sessions) != 2 {
		t.Fatalf("group not saved correctly: %+v", saved.Groups)
	}
}

func TestRunSave_InteractiveSelectsSubset(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	seedActive(t, p, "s1", "s2", "s3")

	var out bytes.Buffer
	r := bufio.NewReader(strings.NewReader("1,3\n"))
	err := RunSave(&out, r, p, "mygroup", false, false, time.Now())
	if err != nil {
		t.Fatalf("RunSave() error = %v", err)
	}

	saved, _, err := registry.LoadSaved(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Groups["mygroup"].Sessions) != 2 {
		t.Fatalf("expected 2 sessions selected, got %+v", saved.Groups["mygroup"])
	}
}

func TestRunSave_DuplicateNameRefusedWithoutForce(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	seedActive(t, p, "s1")

	var out bytes.Buffer
	if err := RunSave(&out, bufio.NewReader(strings.NewReader("")), p, "dup", true, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	err := RunSave(&out, bufio.NewReader(strings.NewReader("")), p, "dup", true, false, time.Now())
	if err == nil {
		t.Fatalf("expected an error for duplicate name without --force")
	}
}

func TestRunSave_ForceOverwritesExisting(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	seedActive(t, p, "s1")
	var out bytes.Buffer
	if err := RunSave(&out, bufio.NewReader(strings.NewReader("")), p, "dup", true, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	seedActive(t, p, "s2")
	if err := RunSave(&out, bufio.NewReader(strings.NewReader("")), p, "dup", true, true, time.Now()); err != nil {
		t.Fatalf("RunSave() with --force error = %v", err)
	}

	saved, _, err := registry.LoadSaved(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Groups["dup"].Sessions) != 2 {
		t.Fatalf("expected overwrite to include both sessions, got %+v", saved.Groups["dup"])
	}
}

func TestRunSave_NoActiveSessionsErrors(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	var out bytes.Buffer
	err := RunSave(&out, bufio.NewReader(strings.NewReader("")), p, "empty", true, false, time.Now())
	if err == nil {
		t.Fatalf("expected an error when there are no active sessions")
	}
}

func TestRunBookmark_InteractivePicksOne(t *testing.T) {
	p := registry.Paths{ConfigDir: t.TempDir()}
	seedActive(t, p, "s1", "s2")

	var out bytes.Buffer
	r := bufio.NewReader(strings.NewReader("2\n"))
	err := RunBookmark(&out, r, p, "robot-fw", time.Now())
	if err != nil {
		t.Fatalf("RunBookmark() error = %v", err)
	}

	saved, _, err := registry.LoadSaved(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := saved.Bookmarks["robot-fw"]; !ok {
		t.Fatalf("bookmark not saved: %+v", saved.Bookmarks)
	}
}
