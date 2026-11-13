// Command nerv-task is the CLI of the nerv-task plugin marketplace:
// install, configure, init and upgrade.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/war-apps/nerv-kit-by-gentle-ai/selfupgrade"
)

const usage = `nerv-task - task tracking plugin for Claude Code

Usage:
  nerv-task install [--with-teamwork] [--no-configure]
  nerv-task configure [--print] [--json] [--set key=value]
  nerv-task init [--provider <name>] [--import] [--none]
  nerv-task upgrade [--alpha|--rc]

Run "nerv-task <command> -h" for a command's own flags.
`

func main() {
	// Best-effort: clean up a stale "<exe>.old" an earlier Windows
	// upgrade may have left behind (selfupgrade.Replace renames aside
	// rather than over, on Windows). No-op on every other run.
	if exe, err := os.Executable(); err == nil {
		selfupgrade.CleanupStaleReplacement(exe)
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "install":
		return runInstall(rest, stdout, stderr)
	case "configure":
		return runConfigure(rest, stdout, stderr)
	case "init":
		return runInit(rest, stdout, stderr)
	case "upgrade":
		return runUpgrade(rest, stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "nerv-task: unknown command %q\n\n", cmd)
		fmt.Fprint(stderr, usage)
		return 2
	}
}
