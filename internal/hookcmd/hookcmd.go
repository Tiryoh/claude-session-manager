// Package hookcmd implements `csm hook`: parsing a Claude Code hook payload
// from stdin and applying it to the active-session registry.
package hookcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// Payload is the subset of Claude Code's hook stdin JSON that csm reads.
// Unknown fields are ignored by encoding/json by default.
type Payload struct {
	SessionID     string `json:"session_id"`
	Cwd           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	Prompt        string `json:"prompt"`
}

// Run reads one hook payload from r and applies it to the registry at p.
// It never panics and has no error return: csm hook must always exit 0 and
// print nothing, so every failure path here is logged to p.HookLogFile()
// instead of surfaced to the caller.
func Run(r io.Reader, p registry.Paths, now time.Time) {
	var payload Payload
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		logError(p, fmt.Errorf("decode payload: %w", err))
		return
	}
	if payload.SessionID == "" {
		logError(p, fmt.Errorf("payload missing session_id (event=%q)", payload.HookEventName))
		return
	}
	err := registry.MutateActive(p, func(reg *registry.ActiveRegistry) {
		apply(reg, payload, now)
	})
	if err != nil {
		logError(p, fmt.Errorf("mutate active registry: %w", err))
	}
}

func apply(reg *registry.ActiveRegistry, p Payload, now time.Time) {
	switch p.HookEventName {
	case "SessionStart":
		s, ok := reg.Sessions[p.SessionID]
		if !ok {
			s = registry.Session{SessionID: p.SessionID, StartedAt: now}
		}
		s.Cwd = p.Cwd
		s.LastSeen = now
		reg.Sessions[p.SessionID] = s
	case "UserPromptSubmit":
		s, ok := reg.Sessions[p.SessionID]
		if !ok {
			s = registry.Session{SessionID: p.SessionID, Cwd: p.Cwd, StartedAt: now}
		}
		s.LastSeen = now
		if s.Label == "" && p.Prompt != "" {
			s.Label = truncateLabel(p.Prompt)
		}
		reg.Sessions[p.SessionID] = s
	case "SessionEnd":
		delete(reg.Sessions, p.SessionID)
	default:
		if s, ok := reg.Sessions[p.SessionID]; ok {
			s.LastSeen = now
			reg.Sessions[p.SessionID] = s
		}
	}
}

// truncateLabel collapses whitespace (including newlines) into single
// spaces and caps the result at 80 runes so picker output stays one line.
func truncateLabel(prompt string) string {
	s := strings.Join(strings.Fields(prompt), " ")
	r := []rune(s)
	if len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return s
}

func logError(p registry.Paths, err error) {
	if mkErr := os.MkdirAll(p.ConfigDir, 0o755); mkErr != nil {
		return
	}
	f, openErr := os.OpenFile(p.HookLogFile(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if openErr != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %v\n", time.Now().Format(time.RFC3339), err)
}
