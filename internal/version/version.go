// Package version holds nerv-task's own binary version, set at build
// time by goreleaser's ldflags (-X .../internal/version.Binary=<tag>),
// mirroring nerv's own versioning.
package version

import (
	"io/fs"

	"github.com/war-apps/nerv-kit-by-gentle-ai/claudeplugin"
)

// Binary is the released tag this binary was built from (for example
// "v1.0.0"). It stays "dev" for a local, non-release build.
var Binary = "dev"

// Info is the NERV ecosystem version contract: every NERV tool reports
// its binary and plugin versions this way, so `nerv` can report it as
// installed or outdated.
type Info struct {
	Binary string `json:"binary"`
	Plugin string `json:"plugin"`
}

// PluginVersion reads the task plugin's own version from its
// .claude-plugin/plugin.json, in the given filesystem.
func PluginVersion(fsys fs.FS) (string, error) {
	return claudeplugin.PluginVersion(fsys)
}
