package nervtask

import (
	"io/fs"
	"testing"
)

// /task:setup configures this repository's .ai-workflow/task.yaml on its
// own, without nerv-task configure's wizard: the provider, its
// repo-scope keys from its provider.yaml, and a connection check pointing
// at the provider's README. Ported from nerv's task_setup_prose_test.go,
// adapted to nerv-task's own file names (T-S3: task.yaml, not a tasks:
// block of config.yaml).
func TestTaskSetupCommand_ConfiguresTheTaskFileOnly(t *testing.T) {
	data, err := fs.ReadFile(CoreFS(), "commands/setup.md")
	if err != nil {
		t.Fatalf("plugin/task/commands/setup.md is missing: %v", err)
	}
	assertContainsAll(t, "task/commands/setup.md", string(data),
		"description:", "Load the `task` skill first",
		".ai-workflow/task.yaml", "provider.yaml", "`scope: repo`", "`scope: user`",
		"defaults", "installed_plugins.json", ".ai-workflow/providers/task-",
		"tool check", "README.md", "one question at a time", "explicit yes",
		"Never commit",
		// A provider named in the arguments but not installed stops the
		// command; it never falls back to the installed one.
		"names a provider that is not in the list", "Never fall back to\n  another provider")
}
