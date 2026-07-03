// Command csm tracks Claude Code sessions across many parallel terminals so
// they can be saved and reopened later, like restoring a browser tab group.
package main

import (
	"fmt"
	"os"
)

const usage = `csm - Claude Code session manager

Usage:
  csm hook                       (invoked by Claude Code hooks; not for manual use)
  csm install [--print]          install csm hooks into ~/.claude/settings.json
  csm uninstall                  remove csm hooks from ~/.claude/settings.json
  csm list [--json]              show active sessions, groups, and bookmarks
  csm save [name] [--all] [--force]   snapshot active sessions as a named group
  csm bookmark [name]            snapshot one active session
  csm open [name] [--fork]       resume a saved session
  csm clean [--dry-run] [--older-than DURATION]   prune unrestorable entries
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "csm: command not yet implemented:", os.Args[1])
	os.Exit(2)
}
