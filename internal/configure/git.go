package configure

import (
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/war-apps/nerv-kit-by-gentle-ai/aiworkflow"
)

// GitSettings are the worktree offer's settings
// (plugin/task/skills/task/SKILL.md §1, §9). They have built-in
// defaults, equal to nerv's own documented ones
// (internal/config/catalogue.go, docs/configuration.md), overridden
// opportunistically by nerv's own configuration when present -- first
// its global user file, then its repository file, which wins. This is
// never a dependency: nerv-task works standalone with no nerv
// installed at all.
type GitSettings struct {
	BranchPattern    string `json:"branch_pattern" yaml:"branch_pattern"`
	CommitRefPattern string `json:"commit_ref_pattern" yaml:"commit_ref_pattern"`
	// Worktree is one of "ask", "always" or "never" -- never a bool.
	Worktree        string `json:"worktree" yaml:"worktree"`
	WorktreePattern string `json:"worktree_pattern" yaml:"worktree_pattern"`
	BaseBranch      string `json:"base_branch" yaml:"base_branch"`
}

// WorktreeValues are GitSettings.Worktree's only allowed values.
var WorktreeValues = []string{"ask", "always", "never"}

// DefaultGit is nerv-task's built-in git settings: exactly nerv's own
// documented defaults, so the worktree offer behaves the same whether
// or not nerv is actually installed.
func DefaultGit() GitSettings {
	return GitSettings{
		BaseBranch:       "develop",
		Worktree:         "ask",
		WorktreePattern:  ".claude/worktrees/{slug}",
		BranchPattern:    "feature/{prefix}-{id}-{slug}",
		CommitRefPattern: "({PREFIX}-{id})",
	}
}

// nervConfigFile is nerv's own configuration file's name, shared by its
// user-scope and project-scope locations (docs/configuration.md: "same
// schema").
const nervConfigFile = "config.yaml"

// nervUserConfigPath is nerv's own global configuration file: kit
// aiworkflow's user-scope ai-workflow directory, holding "config.yaml"
// (nerv's own tool name in the aiworkflow scheme), not nerv-task's own
// "task.yaml".
func nervUserConfigPath(home, goos string) string {
	return filepath.Join(aiworkflow.UserConfigDir(home, goos), "ai-workflow", nervConfigFile)
}

// nervRepoConfigPath is nerv's own repository configuration file.
func nervRepoConfigPath(root string) string {
	return filepath.Join(root, ".ai-workflow", nervConfigFile)
}

// nervGitDoc is the slice of nerv's own config.yaml this package reads
// opportunistically.
type nervGitDoc struct {
	Git struct {
		BranchPattern    *string `yaml:"branch_pattern"`
		CommitRefPattern *string `yaml:"commit_ref_pattern"`
		Worktree         *string `yaml:"worktree"`
		WorktreePattern  *string `yaml:"worktree_pattern"`
		BaseBranch       *string `yaml:"base_branch"`
	} `yaml:"git"`
}

// loadYAML reads path as YAML into v. A missing file is not an error:
// found is false and v is left untouched.
func loadYAML(path string, v any) (found bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return false, err
	}
	return true, nil
}

// applyGitOverride layers doc's set fields onto got, in place. An
// invalid git.worktree value is ignored (got.Worktree is left
// unchanged): this read is opportunistic, never a dependency, so a
// malformed value in nerv's own file must never break nerv-task.
func applyGitOverride(got *GitSettings, doc nervGitDoc) {
	if doc.Git.BranchPattern != nil {
		got.BranchPattern = *doc.Git.BranchPattern
	}
	if doc.Git.CommitRefPattern != nil {
		got.CommitRefPattern = *doc.Git.CommitRefPattern
	}
	if doc.Git.Worktree != nil && slices.Contains(WorktreeValues, *doc.Git.Worktree) {
		got.Worktree = *doc.Git.Worktree
	}
	if doc.Git.WorktreePattern != nil {
		got.WorktreePattern = *doc.Git.WorktreePattern
	}
	if doc.Git.BaseBranch != nil {
		got.BaseBranch = *doc.Git.BaseBranch
	}
}

// ReadNervGit resolves the git settings the worktree offer uses:
// nerv-task's built-in defaults, overridden opportunistically by
// nerv's own configuration when nerv is also set up on this machine or
// in this repository -- first its global user file
// (<aiworkflow user dir>/ai-workflow/config.yaml), then its repository
// file (<repoRoot>/.ai-workflow/config.yaml), which wins key by key
// over the user file. repoRoot may be "" (no repository context, such
// as a bare `nerv-task configure`): only the user file is then
// consulted. Neither file existing, or neither carrying a git: block,
// is not an error: the built-in defaults apply throughout.
func ReadNervGit(home, goos, repoRoot string) (GitSettings, error) {
	got := DefaultGit()

	var userDoc nervGitDoc
	if _, err := loadYAML(nervUserConfigPath(home, goos), &userDoc); err != nil {
		return GitSettings{}, err
	} else {
		applyGitOverride(&got, userDoc)
	}

	if repoRoot != "" {
		var repoDoc nervGitDoc
		if _, err := loadYAML(nervRepoConfigPath(repoRoot), &repoDoc); err != nil {
			return GitSettings{}, err
		} else {
			applyGitOverride(&got, repoDoc)
		}
	}

	return got, nil
}
