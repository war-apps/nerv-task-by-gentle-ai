// Package upgrade implements `nerv-task upgrade`: resolving and
// installing a newer release of the nerv-task binary (kit selfupgrade),
// then re-running install so the new binary refreshes its own plugin
// bundle and settings registration.
package upgrade

import (
	"context"
	"fmt"
	"os"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"
	"github.com/war-apps/nerv-kit-by-gentle-ai/selfupgrade"
)

// Deps bundles what Upgrade needs.
type Deps struct {
	Client         selfupgrade.Client
	Channel        selfupgrade.Channel
	CurrentVersion string
	GOOS           string
	GOARCH         string
	BinaryName     string
	// Exe is the path of the running executable, replaced in place.
	Exe string
	// Runner re-runs "<Exe> install" after a successful replace.
	Runner runner.Runner
}

// Result reports what Upgrade did.
type Result struct {
	UpToDate bool
	Upgraded bool
	From     string
	To       string
}

// Upgrade resolves the newest release of deps.Channel and, when it is
// newer than deps.CurrentVersion, downloads and verifies it, replaces
// deps.Exe in place, and re-runs "<Exe> install" so the new binary
// refreshes its own plugin bundle and settings registration.
func Upgrade(ctx context.Context, deps Deps) (Result, error) {
	installHint := "see https://github.com/war-apps/nerv-task-by-gentle-ai#install"
	if err := selfupgrade.CheckPlatform(deps.GOOS, deps.GOARCH, deps.BinaryName, installHint); err != nil {
		return Result{}, err
	}

	tag, err := deps.Client.Resolve(ctx, deps.Channel)
	if err != nil {
		return Result{}, fmt.Errorf("resolving the latest release: %w", err)
	}
	if !selfupgrade.IsNewer(tag, deps.CurrentVersion) {
		return Result{UpToDate: true, From: deps.CurrentVersion, To: tag}, nil
	}

	dir, err := os.MkdirTemp("", "nerv-task-upgrade-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)

	archive, err := deps.Client.Download(ctx, tag, deps.GOOS, deps.GOARCH, dir)
	if err != nil {
		return Result{}, fmt.Errorf("downloading %s: %w", tag, err)
	}
	if err := selfupgrade.Replace(archive, deps.Exe, deps.GOOS, deps.BinaryName); err != nil {
		return Result{}, fmt.Errorf("replacing the running binary: %w", err)
	}

	if _, _, exitCode, err := deps.Runner.Run(ctx, deps.Exe, "install"); err != nil {
		return Result{}, fmt.Errorf("re-running install: %w", err)
	} else if exitCode != 0 {
		return Result{}, fmt.Errorf("re-running install exited %d", exitCode)
	}

	return Result{Upgraded: true, From: deps.CurrentVersion, To: tag}, nil
}
