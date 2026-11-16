package configure_test

import (
	"path/filepath"
	"testing"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/configure"
)

func TestDefaultGit_MatchesNerv(t *testing.T) {
	got := configure.DefaultGit()
	want := configure.GitSettings{
		BaseBranch:       "develop",
		Worktree:         "ask",
		WorktreePattern:  ".claude/worktrees/{slug}",
		BranchPattern:    "feature/{prefix}-{id}-{slug}",
		CommitRefPattern: "({PREFIX}-{id})",
	}
	if got != want {
		t.Errorf("DefaultGit() = %+v, want %+v", got, want)
	}
}

func TestReadNervGit_NeitherFile_ReturnsBuiltInDefaults(t *testing.T) {
	home := t.TempDir()
	got, err := configure.ReadNervGit(home, "linux", t.TempDir())
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if got != configure.DefaultGit() {
		t.Errorf("ReadNervGit() = %+v, want the built-in defaults", got)
	}
}

func TestReadNervGit_UserOnly_OverridesDefaults(t *testing.T) {
	home := t.TempDir()
	userPath := filepath.Join(home, ".config", "ai-workflow", "config.yaml")
	if err := writeFile(userPath, "git:\n  base_branch: trunk\n  worktree: always\n"); err != nil {
		t.Fatal(err)
	}

	got, err := configure.ReadNervGit(home, "linux", t.TempDir())
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if got.BaseBranch != "trunk" {
		t.Errorf("BaseBranch = %q, want trunk", got.BaseBranch)
	}
	if got.Worktree != "always" {
		t.Errorf("Worktree = %q, want always", got.Worktree)
	}
	// Everything the user file does not set still falls back to the
	// built-in default.
	if got.CommitRefPattern != configure.DefaultGit().CommitRefPattern {
		t.Errorf("CommitRefPattern = %q, want the built-in default", got.CommitRefPattern)
	}
}

func TestReadNervGit_RepoOnly_OverridesDefaults(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	if err := writeFile(filepath.Join(root, ".ai-workflow", "config.yaml"), "git:\n  base_branch: trunk\n"); err != nil {
		t.Fatal(err)
	}

	got, err := configure.ReadNervGit(home, "linux", root)
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if got.BaseBranch != "trunk" {
		t.Errorf("BaseBranch = %q, want trunk", got.BaseBranch)
	}
	if got.Worktree != configure.DefaultGit().Worktree {
		t.Errorf("Worktree = %q, want the built-in default", got.Worktree)
	}
}

func TestReadNervGit_UserAndRepo_RepoWins(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	userPath := filepath.Join(home, ".config", "ai-workflow", "config.yaml")
	if err := writeFile(userPath, "git:\n  base_branch: trunk\n  worktree: always\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(root, ".ai-workflow", "config.yaml"), "git:\n  base_branch: main\n"); err != nil {
		t.Fatal(err)
	}

	got, err := configure.ReadNervGit(home, "linux", root)
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if got.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want main (repo overrides user)", got.BaseBranch)
	}
	// worktree was only set by the user file, which the repo file does
	// not override: it must still apply.
	if got.Worktree != "always" {
		t.Errorf("Worktree = %q, want always (from the user file, not overridden by repo)", got.Worktree)
	}
}

func TestReadNervGit_InvalidWorktreeValue_IsIgnored(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	if err := writeFile(filepath.Join(root, ".ai-workflow", "config.yaml"), "git:\n  worktree: sometimes\n"); err != nil {
		t.Fatal(err)
	}

	got, err := configure.ReadNervGit(home, "linux", root)
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if got.Worktree != configure.DefaultGit().Worktree {
		t.Errorf("Worktree = %q, want the built-in default when the override is invalid", got.Worktree)
	}
}

func TestReadNervGit_NoRepoRoot_StillReadsUser(t *testing.T) {
	home := t.TempDir()
	userPath := filepath.Join(home, ".config", "ai-workflow", "config.yaml")
	if err := writeFile(userPath, "git:\n  base_branch: trunk\n"); err != nil {
		t.Fatal(err)
	}

	got, err := configure.ReadNervGit(home, "linux", "")
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if got.BaseBranch != "trunk" {
		t.Errorf("BaseBranch = %q, want trunk", got.BaseBranch)
	}
}
