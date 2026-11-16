package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"

	nervtask "github.com/war-apps/nerv-task-by-gentle-ai"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/install"
)

const installUsage = `Usage: nerv-task install [--with-teamwork] [--no-configure]

Materializes and registers the nerv-task plugin marketplace in Claude
Code: the task plugin, and the task-teamwork provider when
--with-teamwork is given. When nerv's own task@nerv or
task-teamwork@nerv plugin is enabled, this disables it (never touching
any other nerv plugin), so /task:* is not duplicated.

On a terminal, offers to run nerv-task configure's interactive wizard
once installed. --no-configure skips that offer entirely; without a
terminal, it prints how to run it later instead of prompting.
`

func runInstall(args []string, stdout, stderr io.Writer) int {
	return runInstallWithOptions(args, stdout, stderr, defaultCLIOptions())
}

func runInstallWithOptions(args []string, stdout, stderr io.Writer, opts cliOptions) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	withTeamwork := fs.Bool("with-teamwork", false, "")
	noConfigure := fs.Bool("no-configure", false, "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprint(stderr, installUsage)
		return 2
	}

	home, err := runner.HomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task install: %v\n", err)
		return 2
	}

	taskRunner := opts.Runner
	if taskRunner == nil {
		taskRunner = runner.ExecRunner{}
	}
	deps := install.Deps{
		Home:           home,
		SettingsPath:   filepath.Join(home, ".claude", "settings.json"),
		MarketplaceDir: filepath.Join(home, ".nerv-task", "marketplace"),
		CoreFS:         nervtask.CoreFS(),
		ExtensionsFS:   nervtask.ExtensionsFS(),
		Runner:         taskRunner,
		Now:            time.Now(),
		Stdout:         stdout,
		WithTeamwork:   *withTeamwork,
	}

	result, err := install.Install(deps)
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task install: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "nerv-task installed.")
	for _, id := range result.DisabledLegacyPlugins {
		fmt.Fprintf(stdout, "Disabled %s so /task:* is not duplicated with nerv-task's own plugin.\n", id)
	}

	offerConfigure(home, *noConfigure, stdout, stderr, opts)

	fmt.Fprintln(stdout, "Restart Claude Code for the change to take effect.")
	return 0
}

// offerConfigure is install's end-of-run configuration step: on a
// terminal (and unless --no-configure), asks whether to run nerv-task
// configure's interactive wizard right now; without a terminal, prints
// how to run it later instead of prompting.
func offerConfigure(home string, noConfigure bool, stdout, stderr io.Writer, opts cliOptions) {
	if noConfigure {
		return
	}
	if opts.IsTerminal == nil || !opts.IsTerminal() || opts.Stdin == nil {
		fmt.Fprintln(stdout, "Run `nerv-task configure` to set your provider and personal settings.")
		return
	}

	fmt.Fprint(stdout, "Run nerv-task configure now? [Y/n]: ")
	line, _ := readAnswerLine(opts.Stdin)
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "" && answer != "y" && answer != "yes" {
		return
	}

	deps := buildConfigureDeps(home, stdout)
	runConfigureWizard(deps, opts.Stdin, stdout, stderr)
}
