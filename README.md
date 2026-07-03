# csm — claude-session-manager

Track every open Claude Code session and reopen it later, like restoring a
browser's tab group. Built for running many `claude` sessions in parallel
across terminals: if the machine loses power or a terminal is closed by
accident, `csm` still knows which sessions were open and where.

## How it works

Claude Code hooks (`SessionStart`, `UserPromptSubmit`, `SessionEnd`) call
`csm hook` on every session event. `csm` keeps a registry of active sessions
at `~/.config/claude-session-manager/active.json`. If a session ends
normally, its entry is removed. If the machine loses power, `SessionEnd`
never fires — the entry just sits there, which is exactly the recovery
target `csm` is for.

## Install

```bash
go install ./cmd/csm
csm install
```

`csm install` adds three hook entries to `~/.claude/settings.json`,
preserving every other setting untouched, and writes a `.bak` backup first.
Run it again any time — it's idempotent. `csm uninstall` removes them.

## Day to day

```bash
csm list                          # see active sessions, saved groups, bookmarks
csm save sprint-23                # snapshot the current active sessions as a group (interactive picker)
csm save --all --force autosave   # snapshot everything without prompting -- good for cron
csm bookmark robot-fw             # snapshot one session by picking it interactively
csm open                          # pick anything saved and resume it
csm open sprint-23                # open (picking a session from) a specific group
csm open robot-fw --fork          # resume a bookmark as a fork, leaving the original session intact
csm clean --dry-run               # see which active entries are no longer restorable
```

### Guarding against power loss

Run `csm save --all --force autosave` periodically, e.g. via cron:

```cron
*/15 * * * * PATH=$PATH:/usr/local/go/bin csm save --all --force autosave
```

If the machine loses power, `csm open autosave` lists every session that was
open at the last autosave, ready to resume one by one.

## Scope and caveats

Linux/macOS only. No daemon, no tmux integration — `csm open` resumes one
session at a time by `cd`-ing into its directory and exec'ing
`claude --resume <id>` directly (`syscall.Exec`, replacing the current
process). Because that bypasses the shell, any shell function/alias/flag you
normally wrap `claude` in (e.g. a `claude() { command claude --foo "$@"; }`
function) is **not** applied — `csm open` always launches the plain
`claude` binary found on `PATH`.
