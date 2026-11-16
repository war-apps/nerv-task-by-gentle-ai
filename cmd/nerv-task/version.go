package main

import (
	"encoding/json"
	"fmt"
	"io"

	nervtask "github.com/war-apps/nerv-task-by-gentle-ai"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/version"
)

const versionUsage = `Usage: nerv-task version [--json]

Prints the binary and task plugin versions, following the NERV
ecosystem version contract so "nerv" can report nerv-task as installed
or outdated.

--json   Print the versions as JSON ({"binary":..., "plugin":...}).
`

func runVersion(args []string, stdout, stderr io.Writer) int {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprint(stderr, versionUsage)
			return 2
		}
	}

	pluginVersion, err := version.PluginVersion(nervtask.CoreFS())
	if err != nil {
		fmt.Fprintf(stderr, "nerv-task: %v\n", err)
		return 2
	}

	if jsonOutput {
		info := version.Info{Binary: version.Binary, Plugin: pluginVersion}
		encoded, err := json.Marshal(info)
		if err != nil {
			fmt.Fprintf(stderr, "nerv-task: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "%s\n", encoded)
		return 0
	}

	fmt.Fprintf(stdout, "nerv-task %s (plugin %s)\n", version.Binary, pluginVersion)
	return 0
}
