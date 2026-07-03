// Command csm tracks Claude Code sessions across many parallel terminals so
// they can be saved and reopened later, like restoring a browser tab group.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/hookcmd"
	"github.com/Tiryoh/claude-session-manager/internal/install"
	"github.com/Tiryoh/claude-session-manager/internal/registry"
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

	switch os.Args[1] {
	case "hook":
		runHook()
	case "install":
		runInstall(os.Args[2:])
	case "uninstall":
		runUninstall()
	default:
		fmt.Fprintln(os.Stderr, "csm: command not yet implemented:", os.Args[1])
		os.Exit(2)
	}
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home + "/.claude/settings.json", nil
}

func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	printOnly := fs.Bool("print", false, "print the merged hooks section without writing")
	fs.Parse(args)

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm install: resolve executable path:", err)
		os.Exit(1)
	}
	path, err := settingsPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm install:", err)
		os.Exit(1)
	}

	rendered, changed, err := install.Install(path, exePath, *printOnly)
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm install:", err)
		os.Exit(1)
	}
	if *printOnly {
		os.Stdout.Write(rendered)
		return
	}
	if changed {
		fmt.Println("csm: installed hooks into", path, "(backup at", path+".bak)")
	} else {
		fmt.Println("csm: hooks already installed in", path)
	}
}

func runUninstall() {
	path, err := settingsPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm uninstall:", err)
		os.Exit(1)
	}
	_, changed, err := install.Uninstall(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm uninstall:", err)
		os.Exit(1)
	}
	if changed {
		fmt.Println("csm: removed hooks from", path, "(backup at", path+".bak)")
	} else {
		fmt.Println("csm: no csm hooks were installed in", path)
	}
}

// runHook implements `csm hook`. Per the global constraint, it never writes
// to stdout and always exits 0 -- hookcmd.Run already swallows all errors
// into the hook log, so there is nothing left to check here.
func runHook() {
	paths, err := registry.DefaultPaths()
	if err != nil {
		os.Exit(0)
	}
	hookcmd.Run(os.Stdin, paths, time.Now())
	os.Exit(0)
}
