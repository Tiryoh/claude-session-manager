// Command csm tracks Claude Code sessions across many parallel terminals so
// they can be saved and reopened later, like restoring a browser tab group.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/Tiryoh/claude-session-manager/internal/claudedir"
	"github.com/Tiryoh/claude-session-manager/internal/cli"
	"github.com/Tiryoh/claude-session-manager/internal/hookcmd"
	"github.com/Tiryoh/claude-session-manager/internal/install"
	"github.com/Tiryoh/claude-session-manager/internal/procscan"
	"github.com/Tiryoh/claude-session-manager/internal/registry"
)

// version is set at link time via -ldflags "-X main.version=...".
// Defaults to "dev" for local go build / go install.
var version = "dev"

const usage = `csm - Claude Code session manager

Usage:
  csm --version                  print the version and exit
  csm hook                       (invoked by Claude Code hooks; not for manual use)
  csm install [--print]          install csm hooks into ~/.claude/settings.json
  csm uninstall                  remove csm hooks from ~/.claude/settings.json
  csm list [--json] [--first-message]   show active sessions, groups, and bookmarks
                                  (shows each session's most recent message
                                  by default; --first-message shows its first)
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
	case "-h", "--help", "help":
		fmt.Print(usage)
		os.Exit(0)
	case "-version", "--version", "version":
		fmt.Println(version)
		os.Exit(0)
	case "hook":
		runHook()
	case "install":
		runInstall(os.Args[2:])
	case "uninstall":
		runUninstall()
	case "list":
		runList(os.Args[2:])
	case "save":
		runSave(os.Args[2:])
	case "bookmark":
		runBookmark(os.Args[2:])
	case "open":
		runOpen(os.Args[2:])
	case "clean":
		runClean(os.Args[2:])
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
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

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

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "print machine-readable JSON")
	firstMessage := fs.Bool("first-message", false, "show each session's first message instead of its most recent")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	paths, err := registry.DefaultPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm list:", err)
		os.Exit(1)
	}
	claudeDir, err := claudedir.DefaultClaudeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm list:", err)
		os.Exit(1)
	}
	if err := cli.RunList(os.Stdout, paths, claudeDir, *jsonOut, *firstMessage, procscan.Scan, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "csm list:", err)
		os.Exit(1)
	}
}

func runSave(args []string) {
	fs := flag.NewFlagSet("save", flag.ExitOnError)
	all := fs.Bool("all", false, "save every active session without prompting")
	force := fs.Bool("force", false, "overwrite an existing group with the same name")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	name := ""
	if fs.NArg() > 0 {
		name = fs.Arg(0)
	}

	paths, err := registry.DefaultPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm save:", err)
		os.Exit(1)
	}
	if err := cli.RunSave(os.Stdout, bufio.NewReader(os.Stdin), paths, name, *all, *force, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "csm save:", err)
		os.Exit(1)
	}
}

func runBookmark(args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	paths, err := registry.DefaultPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm bookmark:", err)
		os.Exit(1)
	}
	if err := cli.RunBookmark(os.Stdout, bufio.NewReader(os.Stdin), paths, name, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "csm bookmark:", err)
		os.Exit(1)
	}
}

func runOpen(args []string) {
	fs := flag.NewFlagSet("open", flag.ExitOnError)
	fork := fs.Bool("fork", false, "resume as a fork, leaving the original session untouched")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	name := ""
	if fs.NArg() > 0 {
		name = fs.Arg(0)
	}

	paths, err := registry.DefaultPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm open:", err)
		os.Exit(1)
	}
	claudeDir, err := claudedir.DefaultClaudeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm open:", err)
		os.Exit(1)
	}
	isTTY := stdinIsTerminal()

	err = cli.RunOpen(os.Stdout, bufio.NewReader(os.Stdin), paths, claudeDir, name, *fork, isTTY, execReplace, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm open:", err)
		os.Exit(1)
	}
}

// execReplace changes into dir and replaces the current process with argv,
// so the caller's terminal becomes the resumed Claude session.
func execReplace(dir string, argv []string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found on PATH: %w", err)
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	return syscall.Exec(claudePath, argv, os.Environ())
}

func runClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "report what would be removed without changing anything")
	olderThan := fs.Duration("older-than", 720*time.Hour, "prune idle sessions past this age (default 30 days)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	paths, err := registry.DefaultPaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm clean:", err)
		os.Exit(1)
	}
	claudeDir, err := claudedir.DefaultClaudeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "csm clean:", err)
		os.Exit(1)
	}
	if err := cli.RunClean(os.Stdout, paths, claudeDir, *olderThan, *dryRun, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "csm clean:", err)
		os.Exit(1)
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
