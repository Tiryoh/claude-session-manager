package registry

import (
	"path/filepath"
	"testing"
)

func TestDefaultPaths_UsesEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CSM_CONFIG_DIR", dir)

	p, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}
	if p.ConfigDir != dir {
		t.Fatalf("ConfigDir = %q, want %q", p.ConfigDir, dir)
	}
}

func TestPaths_FileHelpers(t *testing.T) {
	p := Paths{ConfigDir: "/tmp/example"}
	cases := map[string]string{
		p.ActiveFile():  filepath.Join("/tmp/example", "active.json"),
		p.SavedFile():   filepath.Join("/tmp/example", "saved.json"),
		p.HookLogFile(): filepath.Join("/tmp/example", "hook.log"),
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
