package nervtask

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

// Every place that lists the dispatcher's result values names exactly
// provider.Results, in order. Ported from nerv's
// task_events_regress_test.go (TestTaskEnvelope_ResultListsMatchContract),
// scoped to this repository's own skill (no shared nerv-artifacts.md
// here: nerv-task ships no orchestrator).
func TestTaskEnvelope_ResultListsMatchContract(t *testing.T) {
	want := strings.Join(provider.Results, " | ")
	listRe := regexp.MustCompile(`ok \| skipped \| blocked \| failed[^\n]*`)

	data, err := fs.ReadFile(CoreFS(), "skills/task/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	matches := listRe.FindAllString(string(data), -1)
	if len(matches) == 0 {
		t.Fatal("skills/task/SKILL.md never lists the dispatcher's result values")
	}
	for _, m := range matches {
		if !strings.Contains(m, want) {
			t.Errorf("skills/task/SKILL.md lists results %q, want %q", m, want)
		}
	}
}

// The dispatcher's own contract (its agent file plus the task skill it
// loads) names every pipeline event of provider.Events. Ported from
// nerv's TestOrchestrator_EmitsEveryTaskEvent, adapted: nerv-task has no
// orchestrator skill of its own, so this checks the dispatcher's own
// documentation instead.
func TestDispatcher_NamesEveryEvent(t *testing.T) {
	var text strings.Builder
	for _, path := range []string{"agents/dispatcher.md", "skills/task/SKILL.md", "nerv-events.yaml"} {
		data, err := fs.ReadFile(CoreFS(), path)
		if err != nil {
			t.Fatal(err)
		}
		text.Write(data)
	}
	all := text.String()
	assertContainsAll(t, "task:dispatcher contract", all, "task:dispatcher", "task-events.yaml", "/task:replay")

	events := make([]string, 0, len(provider.Events))
	for event := range provider.Events {
		events = append(events, event)
	}
	sort.Strings(events)
	for _, event := range events {
		if !strings.Contains(all, "`"+event+"`") && !strings.Contains(all, "- "+event) {
			t.Errorf("the dispatcher contract never names the event %s", event)
		}
	}
}

// The task contract keeps the testing move tied to the work unit's
// verification in the DIRECT and LIGHT tiers, and never completes a task
// on run.closed. Ported from nerv's
// TestTaskContract_TestingMoveFollowsWorkUnitVerification.
func TestTaskContract_TestingMoveFollowsWorkUnitVerification(t *testing.T) {
	const path = "skills/task/SKILL.md"
	data, err := fs.ReadFile(CoreFS(), path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	assertContainsAll(t, path, string(data), "the work unit's check (LIGHT, DIRECT) starts", "**It never calls `complete`.**")
}
