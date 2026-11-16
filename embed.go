// Package nervtask embeds the plugin marketplace nerv-task ships: the
// task plugin (commands, agent, skill, event contract) and its
// extensions (today, task-teamwork).
package nervtask

import (
	"embed"
	"io/fs"
)

//go:embed all:plugin/task
var coreFS embed.FS

//go:embed all:plugin/extensions
var extensionsFS embed.FS

// CoreFS is the task plugin's own tree, rooted at its own files
// (.claude-plugin/plugin.json, agents/, commands/, skills/,
// nerv-events.yaml).
func CoreFS() fs.FS {
	sub, err := fs.Sub(coreFS, "plugin/task")
	if err != nil {
		panic(err)
	}
	return sub
}

// ExtensionsFS is the tree of every extension plugin nerv-task ships,
// one top-level folder per plugin (today, task-teamwork), each rooted at
// its own files.
func ExtensionsFS() fs.FS {
	sub, err := fs.Sub(extensionsFS, "plugin/extensions")
	if err != nil {
		panic(err)
	}
	return sub
}
