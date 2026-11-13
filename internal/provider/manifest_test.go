package provider_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

const validManifest = `
name: acme
contract_version: 1
runtime: agent
task_ref_prefix: AC
mcp:
  server: acme
  tools:
    - acme-get_task
    - acme-assign
    - acme-move_stage
  connectors: [Acme.com]
config:
  - key: assignee_id
    scope: user
    type: int
    prompt: Your Acme user id
  - key: project_id
    scope: repo
    type: int
    prompt: Acme project id of this repository
stages:
  pending: PENDING
  inDev: IN PROGRESS
operations:
  getTask: procedures/get-task.md
  assign: procedures/assign.md
  moveStage: procedures/move-stage.md
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
events:
  - run.started
`

func TestParse_ValidManifest(t *testing.T) {
	m, err := provider.Parse([]byte(validManifest))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if m.Name != "acme" {
		t.Errorf("Name = %q, want acme", m.Name)
	}
	if m.TaskRefPrefix != "AC" {
		t.Errorf("TaskRefPrefix = %q, want AC", m.TaskRefPrefix)
	}
	if m.MCP.Server != "acme" {
		t.Errorf("MCP.Server = %q, want acme", m.MCP.Server)
	}
	if !m.Implements("getTask") {
		t.Error("Implements(getTask) = false, want true")
	}
	if m.Implements("logTime") {
		t.Error("Implements(logTime) = true, want false (not_implemented)")
	}
	if !m.Subscribes("run.started") {
		t.Error("Subscribes(run.started) = false, want true")
	}
	if m.Subscribes("run.closed") {
		t.Error("Subscribes(run.closed) = true, want false")
	}
}

func TestParse_RejectsBadTaskRefPrefix(t *testing.T) {
	bad := strings.Replace(validManifest, "task_ref_prefix: AC", "task_ref_prefix: ac", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want a task_ref_prefix refusal")
	}
	if !strings.Contains(err.Error(), "task_ref_prefix") {
		t.Errorf("Parse() error = %v, want it to mention task_ref_prefix", err)
	}
}

func TestParse_RejectsUnknownStageKey(t *testing.T) {
	bad := strings.Replace(validManifest, "stages:\n  pending: PENDING", "stages:\n  bogus: X", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want a stages refusal")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("Parse() error = %v, want it to name bogus", err)
	}
}

func TestParse_RejectsDuplicateTool(t *testing.T) {
	bad := strings.Replace(validManifest, "    - acme-assign\n", "    - acme-assign\n    - acme-assign\n", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want a duplicate tool refusal")
	}
	if !strings.Contains(err.Error(), "acme-assign") {
		t.Errorf("Parse() error = %v, want it to name the duplicate tool", err)
	}
}

func TestParse_RejectsDuplicateConfigKey(t *testing.T) {
	bad := strings.Replace(validManifest,
		"  - key: project_id\n    scope: repo\n    type: int\n    prompt: Acme project id of this repository\n",
		"  - key: project_id\n    scope: repo\n    type: int\n    prompt: Acme project id of this repository\n  - key: project_id\n    scope: repo\n    type: int\n    prompt: dup\n",
		1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want a duplicate config key refusal")
	}
	if !strings.Contains(err.Error(), "project_id") {
		t.Errorf("Parse() error = %v, want it to name the duplicate key", err)
	}
}

func TestParse_RejectsBadConfigScope(t *testing.T) {
	bad := strings.Replace(validManifest, "    scope: user\n    type: int\n    prompt: Your Acme user id", "    scope: bogus\n    type: int\n    prompt: Your Acme user id", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want a scope refusal")
	}
	if !strings.Contains(err.Error(), "scope") {
		t.Errorf("Parse() error = %v, want it to mention scope", err)
	}
}

func TestParse_RejectsBadConfigType(t *testing.T) {
	bad := strings.Replace(validManifest, "    type: int\n    prompt: Your Acme user id", "    type: bogus\n    prompt: Your Acme user id", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want a type refusal")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf("Parse() error = %v, want it to mention type", err)
	}
}

func TestParse_RejectsEventNeedingNotImplementedOperation(t *testing.T) {
	bad := strings.Replace(validManifest, "events:\n  - run.started", "events:\n  - run.started\n  - run.closed", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want a refusal: run.closed needs logTime, declared not_implemented")
	}
	if !strings.Contains(err.Error(), "run.closed") || !strings.Contains(err.Error(), "logTime") {
		t.Errorf("Parse() error = %v, want it to name run.closed and logTime", err)
	}
}

func TestParse_RejectsUnknownEvent(t *testing.T) {
	bad := strings.Replace(validManifest, "events:\n  - run.started", "events:\n  - run.started\n  - bogus.event", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want an unknown event refusal")
	}
	if !strings.Contains(err.Error(), "bogus.event") {
		t.Errorf("Parse() error = %v, want it to name bogus.event", err)
	}
}

func TestParse_RejectsBadOperationPath(t *testing.T) {
	bad := strings.Replace(validManifest, "getTask: procedures/get-task.md", "getTask: ../escape.md", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want an operation path refusal")
	}
	if !strings.Contains(err.Error(), "getTask") {
		t.Errorf("Parse() error = %v, want it to name getTask", err)
	}
}

func TestParse_RejectsMissingMCPServerWhenToolsDeclared(t *testing.T) {
	bad := strings.Replace(validManifest, "mcp:\n  server: acme\n", "mcp:\n  server: \"\"\n", 1)
	_, err := provider.Parse([]byte(bad))
	if err == nil {
		t.Fatal("Parse() error = nil, want an mcp.server refusal")
	}
	if !strings.Contains(err.Error(), "mcp.server") {
		t.Errorf("Parse() error = %v, want it to mention mcp.server", err)
	}
}

func TestManifest_CheckFiles_ReportsMissingOperationFile(t *testing.T) {
	m, err := provider.Parse([]byte(validManifest))
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{
		"procedures/get-task.md": &fstest.MapFile{Data: []byte("x")},
		"procedures/assign.md":   &fstest.MapFile{Data: []byte("x")},
		// procedures/move-stage.md intentionally missing.
	}
	err = m.CheckFiles(fsys)
	if err == nil {
		t.Fatal("CheckFiles() error = nil, want a refusal naming move-stage.md")
	}
	if !strings.Contains(err.Error(), "move-stage.md") {
		t.Errorf("CheckFiles() error = %v, want it to name move-stage.md", err)
	}
}

func TestLoad_ReadsManifestFromDir(t *testing.T) {
	fsys := fstest.MapFS{
		"task-acme/provider.yaml":          &fstest.MapFile{Data: []byte(validManifest)},
		"task-acme/procedures/get-task.md": &fstest.MapFile{Data: []byte("x")},
	}
	m, err := provider.Load(fsys, "task-acme")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if m.Name != "acme" {
		t.Errorf("Name = %q, want acme", m.Name)
	}
}

func TestLoad_MissingManifest(t *testing.T) {
	fsys := fstest.MapFS{}
	_, err := provider.Load(fsys, "task-acme")
	if err == nil {
		t.Fatal("Load() error = nil, want a missing-manifest refusal")
	}
}
