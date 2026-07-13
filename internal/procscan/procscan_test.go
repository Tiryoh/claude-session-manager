package procscan

import "testing"

func TestSnapshot_IsRunning(t *testing.T) {
	snap := Snapshot{
		SessionIDs: map[string]bool{"resumed-id": true},
		Cwds:       map[string]bool{"/proj/fresh": true},
	}

	cases := []struct {
		name      string
		sessionID string
		cwd       string
		want      bool
	}{
		{"matches by resumed session id", "resumed-id", "/some/other/dir", true},
		{"matches by cwd for a fresh session", "unknown-id", "/proj/fresh", true},
		{"matches neither", "unknown-id", "/proj/gone", false},
	}
	for _, c := range cases {
		if got := snap.IsRunning(c.sessionID, c.cwd); got != c.want {
			t.Errorf("%s: IsRunning() = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSnapshot_ZeroValueNeverRunning(t *testing.T) {
	var snap Snapshot
	if snap.IsRunning("any-id", "/any/dir") {
		t.Fatal("zero-value Snapshot must report nothing as running")
	}
}
