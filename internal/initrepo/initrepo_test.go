package initrepo_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/initrepo"
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

func testDeps(t *testing.T) initrepo.Deps {
	t.Helper()
	return initrepo.Deps{
		RepoRoot:   t.TempDir(),
		GlobalPath: filepath.Join(t.TempDir(), "task.yaml"),
		Providers:  []*provider.Manifest{testManifest(t)},
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
}

func TestInit_None_WritesNothing(t *testing.T) {
	deps := testDeps(t)
	result, err := initrepo.Init(deps, initrepo.Options{None: true})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !result.Skipped {
		t.Error("Skipped = false, want true")
	}
	if _, err := os.Stat(config.RepoPath(deps.RepoRoot)); err == nil {
		t.Error("task.yaml was written, want nothing for --none")
	}
}

func TestInit_SeedsFromGlobalDefaults(t *testing.T) {
	deps := testDeps(t)
	global := config.GlobalConfig{
		Defaults: &config.RepoConfig{
			Provider: "teamwork",
			Providers: map[string]map[string]any{
				"teamwork": {"project_id": 42},
			},
		},
	}
	if _, _, err := config.SaveGlobal(deps.GlobalPath, global, deps.Now()); err != nil {
		t.Fatal(err)
	}

	if _, err := initrepo.Init(deps, initrepo.Options{}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, found, err := config.LoadRepo(config.RepoPath(deps.RepoRoot))
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("task.yaml was not written")
	}
	if got.Provider != "teamwork" {
		t.Errorf("Provider = %q, want teamwork", got.Provider)
	}
	if got.ProviderValue("teamwork", "project_id") != 42 {
		t.Errorf("project_id = %v, want 42", got.ProviderValue("teamwork", "project_id"))
	}
}

func TestInit_ExplicitProviderOverridesSeed(t *testing.T) {
	deps := testDeps(t)
	if _, err := initrepo.Init(deps, initrepo.Options{Provider: "teamwork"}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	got, found, err := config.LoadRepo(config.RepoPath(deps.RepoRoot))
	if err != nil || !found {
		t.Fatalf("LoadRepo() found=%v err=%v", found, err)
	}
	if got.Provider != "teamwork" {
		t.Errorf("Provider = %q, want teamwork", got.Provider)
	}
}

func TestInit_Import_DropsTasksPrefixAndSplitsScopes(t *testing.T) {
	deps := testDeps(t)
	nervConfig := "tasks:\n" +
		"  provider: teamwork\n" +
		"  ask_when_missing: false\n" +
		"  providers:\n" +
		"    teamwork:\n" +
		"      project_id: 7\n" +
		"      assignee_id: 100\n"
	path := filepath.Join(deps.RepoRoot, ".ai-workflow", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(nervConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := initrepo.Init(deps, initrepo.Options{Import: true})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	repo, found, err := config.LoadRepo(config.RepoPath(deps.RepoRoot))
	if err != nil || !found {
		t.Fatalf("LoadRepo() found=%v err=%v", found, err)
	}
	if repo.Provider != "teamwork" {
		t.Errorf("Provider = %q, want teamwork", repo.Provider)
	}
	if repo.AskWhenMissing == nil || *repo.AskWhenMissing {
		t.Errorf("AskWhenMissing = %v, want false", repo.AskWhenMissing)
	}
	if repo.ProviderValue("teamwork", "project_id") != 7 {
		t.Errorf("repo project_id = %v, want 7 (scope: repo)", repo.ProviderValue("teamwork", "project_id"))
	}
	if repo.ProviderValue("teamwork", "assignee_id") != nil {
		t.Errorf("repo assignee_id = %v, want unset (scope: user belongs in the global file)", repo.ProviderValue("teamwork", "assignee_id"))
	}

	// Security: a repository's own tasks: block is untrusted input. A
	// scope: user key found there (assignee_id) must never be written
	// to the global file, even when the global file does not set it
	// yet: that would poison the global config -- shared by every other
	// repository -- from one repo's committed file.
	if _, found, err := config.LoadGlobal(deps.GlobalPath); err != nil {
		t.Fatal(err)
	} else if found {
		t.Error("the global task.yaml was written by an import; it must stay untouched")
	}

	wantSkip := "teamwork.assignee_id is personal; set it with `nerv-task configure --set teamwork.assignee_id=<value>`"
	if !containsString(result.SkippedPersonalKeys, wantSkip) {
		t.Errorf("SkippedPersonalKeys = %v, want it to contain %q", result.SkippedPersonalKeys, wantSkip)
	}
}

func TestInit_Import_NeverWritesOrOverwritesGlobalFile(t *testing.T) {
	deps := testDeps(t)
	existingGlobal := config.GlobalConfig{Providers: map[string]map[string]any{
		"teamwork": {"assignee_id": 999},
	}}
	if _, _, err := config.SaveGlobal(deps.GlobalPath, existingGlobal, deps.Now()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(deps.GlobalPath)
	if err != nil {
		t.Fatal(err)
	}

	nervConfig := "tasks:\n  providers:\n    teamwork:\n      assignee_id: 100\n"
	path := filepath.Join(deps.RepoRoot, ".ai-workflow", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(nervConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := initrepo.Init(deps, initrepo.Options{Import: true})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	after, err := os.ReadFile(deps.GlobalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("the global task.yaml changed:\nbefore: %s\nafter:  %s", before, after)
	}
	if len(result.SkippedPersonalKeys) != 1 {
		t.Errorf("SkippedPersonalKeys = %v, want exactly one entry", result.SkippedPersonalKeys)
	}
}

func TestInit_Import_InvalidTypeForRepoScopeKey_SkippedAndReported(t *testing.T) {
	deps := testDeps(t)
	nervConfig := "tasks:\n  providers:\n    teamwork:\n      project_id: \"not-an-int\"\n"
	path := filepath.Join(deps.RepoRoot, ".ai-workflow", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(nervConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := initrepo.Init(deps, initrepo.Options{Import: true})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	repo, _, err := config.LoadRepo(config.RepoPath(deps.RepoRoot))
	if err != nil {
		t.Fatal(err)
	}
	if repo.ProviderValue("teamwork", "project_id") != nil {
		t.Errorf("project_id = %v, want unset (invalid type must be skipped)", repo.ProviderValue("teamwork", "project_id"))
	}
	if len(result.InvalidKeys) != 1 {
		t.Errorf("InvalidKeys = %v, want exactly one entry", result.InvalidKeys)
	}
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
