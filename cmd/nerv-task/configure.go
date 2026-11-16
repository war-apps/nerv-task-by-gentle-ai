package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"

	nervtask "github.com/war-apps/nerv-task-by-gentle-ai"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/configure"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

const configureUsage = `Usage: nerv-task configure [--print] [--json] [--set key=value] [--answers <file>]

With no mode flag: on a terminal, runs the interactive wizard (pick
which providers to enable, then their personal settings, then
timer_store and stale_after_hours, with a diff and a confirmation
before writing); without a terminal, prints the current configuration
and how to set a key (no prompts).

--print            Print the global task.yaml: its config path, values,
                    defaults seed, known providers and git settings.
--json             Render --print as JSON instead of text.
--set key=value    Set one global key (timer_store, stale_after_hours,
                    rounding_minutes) or one provider's scope: user key
                    (<provider>.<key>). May be given more than once.
--answers <file>   Drive the interactive wizard from a file instead of
                    a real terminal, one answer per line. A missing
                    file is treated as empty.
`

// stringList is a repeatable flag.Value, for --set.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runConfigure(args []string, stdout, stderr io.Writer) int {
	return runConfigureWithOptions(args, stdout, stderr, defaultCLIOptions())
}

func runConfigureWithOptions(args []string, stdout, stderr io.Writer, opts cliOptions) int {
	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	printFlag := fs.Bool("print", false, "")
	jsonFlag := fs.Bool("json", false, "")
	answersFlag := fs.String("answers", "", "")
	var sets stringList
	fs.Var(&sets, "set", "")
	if err := fs.Parse(args); err != nil {
		fmt.Fprint(stderr, configureUsage)
		return 2
	}

	home, err := runner.HomeDir()
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
		return 2
	}
	deps := buildConfigureDeps(home, stdout)

	for _, kv := range sets {
		result, err := configure.Set(deps, kv)
		if err != nil {
			fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s: %s -> %s\n", result.Key, orDash(result.Old), result.New)
	}

	flagMode := len(sets) > 0 || *printFlag || *jsonFlag
	if flagMode {
		if len(sets) > 0 && !*printFlag && !*jsonFlag {
			return 0
		}
		return printConfigure(deps, stdout, stderr, *jsonFlag)
	}

	// No mode flag: the interactive wizard, driven by --answers when
	// given, otherwise the real terminal; without either, print the
	// current configuration and how to set a key (no prompts).
	switch {
	case *answersFlag != "":
		in, err := openAnswersFile(*answersFlag)
		if err != nil {
			fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
			return 2
		}
		return runConfigureWizard(deps, in, stdout, stderr)
	case opts.IsTerminal != nil && opts.IsTerminal():
		return runConfigureWizard(deps, opts.Stdin, stdout, stderr)
	default:
		if code := printConfigure(deps, stdout, stderr, false); code != 0 {
			return code
		}
		fmt.Fprintln(stdout, "\nstdin is not a terminal: set a key with --set key=value, or run on a terminal (or with --answers <file>) to configure interactively.")
		return 0
	}
}

// runConfigureWizard runs the interactive wizard against in, reporting
// ErrInputClosed (a genuinely exhausted reader, such as an --answers
// file that ran out) as a refusal that wrote nothing.
func runConfigureWizard(deps configure.Deps, in io.Reader, stdout, stderr io.Writer) int {
	if _, err := configure.RunWizard(deps, in, stdout); err != nil {
		fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
		return 1
	}
	return 0
}

// openAnswersFile reads path's whole content up front (small text
// files, one answer per line). A missing file is treated as an empty
// answers file -- never an error (mirrors nerv's own configure).
func openAnswersFile(path string) (io.Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return strings.NewReader(""), nil
		}
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func printConfigure(deps configure.Deps, stdout, stderr io.Writer, asJSON bool) int {
	out, err := configure.Print(deps)
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
		return 1
	}
	if asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
			return 1
		}
		return 0
	}
	printText(stdout, out)
	return 0
}

func buildConfigureDeps(home string, stdout io.Writer) configure.Deps {
	return configure.Deps{
		GlobalPath: config.UserPath(home, goos()),
		Providers:  discoverProviders(),
		Git:        resolveGit(home),
		Now:        time.Now,
		Stdout:     stdout,
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func goos() string { return runtime.GOOS }

func discoverProviders() []*provider.Manifest {
	root, err := gitToplevel()
	if err != nil {
		root = ""
	}
	manifests, err := provider.Discover(nervtask.ExtensionsFS(), root)
	if err != nil {
		return nil
	}
	return manifests
}

func resolveGit(home string) configure.GitSettings {
	root, err := gitToplevel()
	if err != nil {
		root = ""
	}
	got, err := configure.ReadNervGit(home, goos(), root)
	if err != nil {
		return configure.DefaultGit()
	}
	return got
}

func gitToplevel() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

func printText(stdout io.Writer, out configure.PrintOutput) {
	fmt.Fprintf(stdout, "config_path: %s\n", out.ConfigPath)
	fmt.Fprintf(stdout, "timer_store: %s\n", orDash(out.Values.TimerStore))
	fmt.Fprintln(stdout, "providers:")
	for _, p := range out.Providers {
		fmt.Fprintf(stdout, "  - %s (task_ref_prefix %s)\n", p.Name, p.TaskRefPrefix)
		for _, c := range p.Config {
			fmt.Fprintf(stdout, "      %s.%s  scope=%s type=%s\n", p.Name, c.Key, c.Scope, c.Type)
		}
	}
	fmt.Fprintln(stdout, "git:")
	fmt.Fprintf(stdout, "  branch_pattern: %s\n", out.Git.BranchPattern)
	fmt.Fprintf(stdout, "  commit_ref_pattern: %s\n", out.Git.CommitRefPattern)
	fmt.Fprintf(stdout, "  worktree: %v\n", out.Git.Worktree)
	fmt.Fprintf(stdout, "  worktree_pattern: %s\n", out.Git.WorktreePattern)
	fmt.Fprintf(stdout, "  base_branch: %s\n", out.Git.BaseBranch)
}
