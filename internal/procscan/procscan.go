// Package procscan tells csm which of its tracked sessions still have a
// live `claude` process behind them, by scanning the OS process table
// instead of asking Claude Code (which exposes no such API).
package procscan

// Snapshot is one scan of the running `claude` processes on the host.
type Snapshot struct {
	// SessionIDs holds every session id found in a `--resume <id>` argv --
	// an exact, unambiguous match for a resumed session.
	SessionIDs map[string]bool
	// Cwds holds the working directory of every running claude process,
	// resumed or not. It's the only signal available for a freshly started
	// (non-resumed) session, whose argv carries no session id at all.
	Cwds map[string]bool
}

// IsRunning reports whether a tracked session with the given id/cwd looks
// alive: either its exact id was found in a --resume argument, or some
// claude process shares its working directory.
func (s Snapshot) IsRunning(sessionID, cwd string) bool {
	if s.SessionIDs[sessionID] {
		return true
	}
	return s.Cwds[cwd]
}
