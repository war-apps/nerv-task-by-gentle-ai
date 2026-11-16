package nervtask

import (
	"io/fs"
	"testing"
)

// A Teamwork task without a workflow has no stage to move to: moveStage
// records the stage change as a task comment instead, so the change is
// still visible on the task. A task whose workflow has no matching stage
// stays skipped. Ported from nerv's teamwork_stage_comment_prose_test.go.
func TestTeamworkMoveStage_CommentsWithoutAWorkflow(t *testing.T) {
	read := func(fsys fs.FS, path string) string {
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		return string(data)
	}
	assertContainsAll(t, "task-teamwork/procedures/move-stage.md",
		read(ExtensionsFS(), "task-teamwork/procedures/move-stage.md"),
		"no workflow", "twprojects-create_comment", "comment_labels.stage", "ESTADO", "recorded as a comment",
		"no stage matching <target>")
	assertContainsAll(t, "task/skills/task/SKILL.md",
		read(CoreFS(), "skills/task/SKILL.md"),
		"`ESTADO`", "recorded as a comment")
}
