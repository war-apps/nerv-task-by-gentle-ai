package initrepo_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/initrepo"
)

// trySymlink creates a symlink, skipping the test (not failing it) when
// the current machine refuses symlink creation for lack of privilege --
// common on Windows without Developer Mode or admin rights.
func trySymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if os.IsPermission(err) {
			t.Skipf("creating symlinks needs privileges on this machine: %v", err)
		}
		t.Fatal(err)
	}
}

// TestInit_SymlinkedAiWorkflowDir_Refused is the directory-escape case:
// a cloned repository ships .ai-workflow itself as a symlink (e.g. to
// the user's global config directory), which would otherwise make init
// write repository-derived, untrusted content there, bypassing the
// personal-key protection entirely.
func TestInit_SymlinkedAiWorkflowDir_Refused(t *testing.T) {
	repo := t.TempDir()
	elsewhere := t.TempDir()
	trySymlink(t, elsewhere, filepath.Join(repo, ".ai-workflow"))

	globalPath := filepath.Join(t.TempDir(), "task.yaml")
	global := config.GlobalConfig{Providers: map[string]map[string]any{
		"teamwork": {"assignee_id": 999},
	}}
	if _, _, err := config.SaveGlobal(globalPath, global, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatal(err)
	}

	deps := initrepo.Deps{
		RepoRoot:   repo,
		GlobalPath: globalPath,
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	if _, err := initrepo.Init(deps, initrepo.Options{}); err == nil {
		t.Fatal("Init() error = nil, want a refusal for a symlinked .ai-workflow directory")
	}

	if _, err := os.Stat(filepath.Join(elsewhere, "task.yaml")); err == nil {
		t.Error("task.yaml was written through the symlink target")
	}
	after, err := os.ReadFile(globalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("the global task.yaml was modified despite the refusal")
	}
}

// TestInit_SymlinkedTaskYAML_Refused is the file-escape case: the
// .ai-workflow directory is real, but task.yaml inside it is a symlink
// to a file outside the repository (such as the user's own global
// task.yaml): writing through it would silently overwrite that file
// with untrusted, repository-derived content.
func TestInit_SymlinkedTaskYAML_Refused(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".ai-workflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "victim.yaml")
	if err := os.WriteFile(target, []byte("provider: original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	trySymlink(t, target, filepath.Join(repo, ".ai-workflow", "task.yaml"))

	deps := initrepo.Deps{
		RepoRoot:   repo,
		GlobalPath: filepath.Join(t.TempDir(), "task.yaml"),
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	if _, err := initrepo.Init(deps, initrepo.Options{}); err == nil {
		t.Fatal("Init() error = nil, want a refusal for a symlinked task.yaml")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "provider: original\n" {
		t.Errorf("victim.yaml = %q, want it untouched", data)
	}
}

// TestInit_Import_SymlinkedNervConfigYAML_Refused: a cloned repository
// must not be able to make --import read through a symlink pointing
// outside the repository.
func TestInit_Import_SymlinkedNervConfigYAML_Refused(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".ai-workflow"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "victim-config.yaml")
	if err := os.WriteFile(target, []byte("tasks:\n  provider: victim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	trySymlink(t, target, filepath.Join(repo, ".ai-workflow", "config.yaml"))

	deps := initrepo.Deps{
		RepoRoot:   repo,
		GlobalPath: filepath.Join(t.TempDir(), "task.yaml"),
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	if _, err := initrepo.Init(deps, initrepo.Options{Import: true}); err == nil {
		t.Fatal("Init() error = nil, want a refusal for a symlinked config.yaml")
	}
}

// TestInit_NormalPaths_StillWork is the regression guard: ordinary,
// non-symlinked repositories must keep working exactly as before.
func TestInit_NormalPaths_StillWork(t *testing.T) {
	repo := t.TempDir()
	deps := initrepo.Deps{
		RepoRoot:   repo,
		GlobalPath: filepath.Join(t.TempDir(), "task.yaml"),
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	result, err := initrepo.Init(deps, initrepo.Options{})
	if err != nil {
		t.Fatalf("Init() error = %v, want normal paths to keep working", err)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Errorf("task.yaml was not written: %v", err)
	}
}
