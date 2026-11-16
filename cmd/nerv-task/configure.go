package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

const configureUsage = `Usage: nerv-task configure [--print] [--json] [--set key=value]

With no flags, prints the current global configuration (same as --print).

--print            Print the global task.yaml: its config path, values,
                    defaults seed, known providers and git settings.
--json             Render --print as JSON instead of text.
--set key=value    Set one global key (timer_store, stale_after_hours,
                    rounding_minutes) or one provider's scope: user key
                    (<provider>.<key>). May be given more than once.
`

// stringList is a repeatable flag.Value, for --set.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runConfigure(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	printFlag := fs.Bool("print", false, "")
	jsonFlag := fs.Bool("json", false, "")
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
	deps := configure.Deps{
		GlobalPath: config.UserPath(home, goos()),
		Providers:  discoverProviders(),
		Git:        resolveGit(home),
		Now:        time.Now,
		Stdin:      os.Stdin,
		Stdout:     stdout,
	}

	for _, kv := range sets {
		result, err := configure.Set(deps, kv)
		if err != nil {
			fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s: %s -> %s\n", result.Key, orDash(result.Old), result.New)
	}

	if len(sets) > 0 && !*printFlag && !*jsonFlag {
		return 0
	}

	out, err := configure.Print(deps)
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task configure: %v\n", err)
		return 1
	}
	if *jsonFlag {
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
