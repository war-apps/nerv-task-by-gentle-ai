package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"
	"github.com/war-apps/nerv-kit-by-gentle-ai/selfupgrade"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/upgrade"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/version"
)

const (
	upgradeAPIBase      = "https://api.github.com/repos/war-apps/nerv-task-by-gentle-ai"
	upgradeDownloadBase = "https://github.com/war-apps/nerv-task-by-gentle-ai/releases/download"
	upgradeBinaryName   = "nerv-task"
)

const upgradeUsage = `Usage: nerv-task upgrade [--alpha|--rc]

Replaces the running nerv-task binary with the newest release, then
re-runs install so the new binary refreshes its own plugin bundle and
settings registration.

--alpha   Resolve the newest alpha pre-release instead of stable.
--rc      Resolve the newest release-candidate pre-release instead of stable.
`

func runUpgrade(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	alpha := fs.Bool("alpha", false, "")
	rc := fs.Bool("rc", false, "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprint(stderr, upgradeUsage)
		return 2
	}
	if *alpha && *rc {
		fmt.Fprintln(stderr, "nerv-task upgrade: --alpha and --rc are mutually exclusive")
		return 2
	}
	channel := selfupgrade.Stable
	switch {
	case *alpha:
		channel = selfupgrade.Alpha
	case *rc:
		channel = selfupgrade.RC
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task upgrade: %v\n", err)
		return 2
	}
	selfupgrade.CleanupStaleReplacement(exe)

	deps := upgrade.Deps{
		Client: selfupgrade.Client{
			HTTP:         http.DefaultClient,
			APIBase:      upgradeAPIBase,
			DownloadBase: upgradeDownloadBase,
			BinaryName:   upgradeBinaryName,
		},
		Channel:        channel,
		CurrentVersion: version.Binary,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		BinaryName:     upgradeBinaryName,
		Exe:            exe,
		Runner:         runner.ExecRunner{},
	}

	result, err := upgrade.Upgrade(context.Background(), deps)
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task upgrade: %v\n", err)
		return 1
	}
	if result.UpToDate {
		fmt.Fprintf(stdout, "nerv-task is already up to date (%s).\n", result.From)
		return 0
	}
	fmt.Fprintf(stdout, "Upgraded nerv-task %s -> %s.\n", result.From, result.To)
	return 0
}
