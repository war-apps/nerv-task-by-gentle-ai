package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/initrepo"
)

const initUsage = `Usage: nerv-task init [--provider <name>] [--import] [--none]

Writes this repository's own .ai-workflow/task.yaml, seeded from the
global task.yaml's defaults: block.

--provider <name>  Set the repository's provider, overriding the seed.
--import           Copy nerv's own .ai-workflow/config.yaml tasks:
                    block into the new file (dropping the tasks.
                    prefix), repo-scope keys only. A repository's own
                    file is untrusted input: personal (scope: user)
                    keys are never written anywhere, including the
                    global task.yaml -- they are reported as skipped,
                    for you to set yourself with nerv-task configure.
--none              Write nothing: task tracking stays off here.
`

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	providerFlag := fs.String("provider", "", "")
	importFlag := fs.Bool("import", false, "")
	noneFlag := fs.Bool("none", false, "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprint(stderr, initUsage)
		return 2
	}

	root, err := gitToplevel()
	if err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			fmt.Fprintf(stderr, "nerv-task init: %v\n", err)
			return 2
		}
		root = wd
	}

	home, err := runner.HomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task init: %v\n", err)
		return 2
	}

	deps := initrepo.Deps{
		RepoRoot:   root,
		GlobalPath: config.UserPath(home, goos()),
		Providers:  discoverProviders(),
		Now:        time.Now,
		Stdin:      os.Stdin,
		Stdout:     stdout,
	}

	result, err := initrepo.Init(deps, initrepo.Options{
		Provider: *providerFlag,
		Import:   *importFlag,
		None:     *noneFlag,
	})
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task init: %v\n", err)
		return 1
	}

	if result.Skipped {
		fmt.Fprintln(stdout, "Task tracking stays off in this repository (--none): nothing written.")
		return 0
	}
	if result.ImportedFrom != "" {
		fmt.Fprintf(stdout, "Imported %s's tasks: block.\n", result.ImportedFrom)
		for _, msg := range result.SkippedPersonalKeys {
			fmt.Fprintf(stdout, "  skipped (personal): %s\n", msg)
		}
		for _, msg := range result.InvalidKeys {
			fmt.Fprintf(stdout, "  skipped (invalid): %s\n", msg)
		}
	}
	if result.SeededFromDefaults {
		fmt.Fprintln(stdout, "Seeded from the global task.yaml's defaults.")
	}
	fmt.Fprintf(stdout, "Wrote %s\n", result.Path)
	return 0
}
