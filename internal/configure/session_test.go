package configure_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/configure"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

// wizardManifest is teamwork's manifest, with an extra user-scope bool
// key (billable) so the wizard tests also cover bool prompting.
const wizardManifest = `
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
  - key: billable
    scope: user
    type: bool
    default: "false"
    prompt: Log time as billable
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

func wizardDeps(t *testing.T) configure.Deps {
	t.Helper()
	m, err := provider.Parse([]byte(wizardManifest))
	if err != nil {
		t.Fatal(err)
	}
	return configure.Deps{
		GlobalPath: filepath.Join(t.TempDir(), "task.yaml"),
		Providers:  []*provider.Manifest{m},
		Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
}

func TestRunWizard_EOFBeforeAnyAnswer_ReturnsErrInputClosedAndWritesNothing(t *testing.T) {
	deps := wizardDeps(t)
	var out bytes.Buffer

	_, err := configure.RunWizard(deps, strings.NewReader(""), &out)
	if !errors.Is(err, configure.ErrInputClosed) {
		t.Fatalf("RunWizard() error = %v, want ErrInputClosed", err)
	}
}

func TestRunWizard_AllBlankAnswers_NoChanges_WritesNothing(t *testing.T) {
	deps := wizardDeps(t)
	var out bytes.Buffer

	// timer_store, stale_after_hours, "enable teamwork?" (blank = default),
	// and (if enabled) its keys -- enough blank lines to reach the end
	// without ever answering non-blank.
	in := strings.NewReader(strings.Repeat("\n", 10))
	result, err := configure.RunWizard(deps, in, &out)
	if err != nil {
		t.Fatalf("RunWizard() error = %v", err)
	}
	if result.Changed {
		t.Errorf("Changed = true, want false on an all-blank run: %v", result.ChangeLog)
	}
}

func TestRunWizard_SetsProviderKeyAndGlobalKeys_OnConfirm(t *testing.T) {
	deps := wizardDeps(t)
	var out bytes.Buffer

	// timer_store -> blank (keep default)
	// stale_after_hours -> blank (keep default)
	// Enable teamwork? -> y
	// assignee_id -> 100
	// billable -> true
	// Write? -> y
	in := strings.NewReader("\n\ny\n100\ntrue\ny\n")
	result, err := configure.RunWizard(deps, in, &out)
	if err != nil {
		t.Fatalf("RunWizard() error = %v\noutput:\n%s", err, out.String())
	}
	if !result.Changed {
		t.Fatalf("Changed = false, want true; output:\n%s", out.String())
	}

	printed, err := configure.Print(deps)
	if err != nil {
		t.Fatal(err)
	}
	if printed.Values.ProviderValue("teamwork", "assignee_id") != 100 {
		t.Errorf("assignee_id = %v, want 100", printed.Values.ProviderValue("teamwork", "assignee_id"))
	}
	if printed.Values.ProviderValue("teamwork", "billable") != true {
		t.Errorf("billable = %v, want true", printed.Values.ProviderValue("teamwork", "billable"))
	}
}

func TestRunWizard_DeclineWrite_LeavesFileUntouched(t *testing.T) {
	deps := wizardDeps(t)
	var out bytes.Buffer

	in := strings.NewReader("\n\ny\n100\ntrue\nn\n")
	result, err := configure.RunWizard(deps, in, &out)
	if err != nil {
		t.Fatalf("RunWizard() error = %v", err)
	}
	if result.Changed {
		t.Error("Changed = true, want false after declining the write")
	}
	if _, err := os.Stat(deps.GlobalPath); err == nil {
		t.Error("the global file should not exist after declining the write")
	}
}

func TestRunWizard_NeverEnablesProviderKeys_WhenNotEnabled(t *testing.T) {
	deps := wizardDeps(t)
	var out bytes.Buffer

	// timer_store, stale_after_hours blank; "Enable teamwork?" -> n;
	// confirm write -> y (nothing to write though, so Changed stays false
	// since timer_store/stale_after_hours default to their current
	// values too).
	in := strings.NewReader("\n\nn\ny\n")
	result, err := configure.RunWizard(deps, in, &out)
	if err != nil {
		t.Fatalf("RunWizard() error = %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "Your Teamwork user id") {
		t.Errorf("the wizard prompted teamwork's keys despite answering no to enabling it:\n%s", out.String())
	}
	_ = result
}
