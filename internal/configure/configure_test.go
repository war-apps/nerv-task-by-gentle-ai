package configure_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/configure"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

const teamworkManifest = `
name: teamwork
contract_version: 1
runtime: agent
task_ref_prefix: TW
mcp:
  server: teamwork
  tools: [twprojects-get_task]
config:
  - key: assignee_id
    scope: user
    type: int
    prompt: Your Teamwork user id
  - key: project_id
    scope: repo
    type: int
    prompt: Teamwork project id of this repository
operations:
  getTask: procedures/get-task.md
  assign: not_implemented
  moveStage: not_implemented
  logTime: not_implemented
  updateTimelog: not_implemented
  deleteTimelog: not_implemented
  listTimelogs: not_implemented
  createTask: not_implemented
  createSubtask: not_implemented
  comment: not_implemented
  complete: not_implemented
  setPriority: not_implemented
  listTasks: not_implemented
events: []
`

func testManifest(t *testing.T) *provider.Manifest {
	t.Helper()
	m, err := provider.Parse([]byte(teamworkManifest))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func testDeps(t *testing.T) configure.Deps {
	t.Helper()
	return configure.Deps{
		GlobalPath: filepath.Join(t.TempDir(), "task.yaml"),
		Providers:  []*provider.Manifest{testManifest(t)},
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
}

func TestPrint_ReportsConfigPathDefaultsAndProviders(t *testing.T) {
	deps := testDeps(t)
	out, err := configure.Print(deps)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if out.ConfigPath != deps.GlobalPath {
		t.Errorf("ConfigPath = %q, want %q", out.ConfigPath, deps.GlobalPath)
	}
	if len(out.Providers) != 1 || out.Providers[0].Name != "teamwork" {
		t.Errorf("Providers = %v, want one entry named teamwork", out.Providers)
	}
	if out.Git.BaseBranch == "" {
		t.Error("Git.BaseBranch is empty, want a built-in default")
	}
}

func TestSet_GlobalScalarKey_PersistsAndReads(t *testing.T) {
	deps := testDeps(t)
	res, err := configure.Set(deps, "timer_store=/custom/timers.json")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if res.New != "/custom/timers.json" {
		t.Errorf("New = %q, want /custom/timers.json", res.New)
	}

	out, err := configure.Print(deps)
	if err != nil {
		t.Fatal(err)
	}
	if out.Values.TimerStore != "/custom/timers.json" {
		t.Errorf("Values.TimerStore = %q, want /custom/timers.json", out.Values.TimerStore)
	}
}

func TestSet_ProviderUserScopeKey_Persists(t *testing.T) {
	deps := testDeps(t)
	if _, err := configure.Set(deps, "teamwork.assignee_id=100"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	out, err := configure.Print(deps)
	if err != nil {
		t.Fatal(err)
	}
	if out.Values.ProviderValue("teamwork", "assignee_id") != 100 {
		t.Errorf("assignee_id = %v, want 100", out.Values.ProviderValue("teamwork", "assignee_id"))
	}
}

func TestSet_ProviderRepoScopeKey_Refused(t *testing.T) {
	deps := testDeps(t)
	_, err := configure.Set(deps, "teamwork.project_id=1")
	if err == nil {
		t.Fatal("Set() error = nil, want a refusal for a scope: repo key")
	}
	if !strings.Contains(err.Error(), "repo") {
		t.Errorf("Set() error = %v, want it to mention the repo scope", err)
	}
}

func TestSet_UnknownProvider_Refused(t *testing.T) {
	deps := testDeps(t)
	_, err := configure.Set(deps, "bogus.key=1")
	if err == nil {
		t.Fatal("Set() error = nil, want a refusal for an unknown provider")
	}
}

func TestSet_UnknownProviderKey_Refused(t *testing.T) {
	deps := testDeps(t)
	_, err := configure.Set(deps, "teamwork.bogus=1")
	if err == nil {
		t.Fatal("Set() error = nil, want a refusal for an unknown key")
	}
}

func TestSet_InvalidIntValue_Refused(t *testing.T) {
	deps := testDeps(t)
	_, err := configure.Set(deps, "teamwork.assignee_id=not-a-number")
	if err == nil {
		t.Fatal("Set() error = nil, want a refusal for an invalid int")
	}
}

func TestSet_MalformedAssignment_Refused(t *testing.T) {
	deps := testDeps(t)
	_, err := configure.Set(deps, "no-equals-sign")
	if err == nil {
		t.Fatal("Set() error = nil, want a refusal for a malformed assignment")
	}
}

func TestReadNervGit_MissingFile_ReturnsFalseNoError(t *testing.T) {
	got, found, err := configure.ReadNervGit(t.TempDir())
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if found {
		t.Errorf("ReadNervGit() found = true, want false; got %+v", got)
	}
}

func TestReadNervGit_ReadsGitBlockWhenPresent(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".ai-workflow")
	if err := writeFile(filepath.Join(dir, "config.yaml"), "git:\n  base_branch: trunk\n  worktree: false\n"); err != nil {
		t.Fatal(err)
	}
	got, found, err := configure.ReadNervGit(root)
	if err != nil {
		t.Fatalf("ReadNervGit() error = %v", err)
	}
	if !found {
		t.Fatal("ReadNervGit() found = false, want true")
	}
	if got.BaseBranch != "trunk" {
		t.Errorf("BaseBranch = %q, want trunk", got.BaseBranch)
	}
	if got.Worktree {
		t.Error("Worktree = true, want false (explicit override)")
	}
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
