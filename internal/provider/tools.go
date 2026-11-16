package provider

import (
	"regexp"
	"slices"
	"strings"
)

var connectorNameChar = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// ConnectorToolPrefix is the tool-name prefix Claude Code gives a
// claude.ai connector named name: every character but letters, digits,
// "_" and "-" reads as "_".
func ConnectorToolPrefix(name string) string {
	return "mcp__claude_ai_" + connectorNameChar.ReplaceAllString(name, "_") + "__"
}

// MissingTools returns the declared tools absent from available, in
// declaration order. A tool counts when available holds its bare name,
// its Claude Code name mcp__<server>__<tool>, or the name Claude Code
// gives it when the provider's own plugin (task-<name>) ships the
// server in its .mcp.json (mcp__plugin_task-<name>_<server>__<tool> or
// the dash-to-underscore variant mcp__plugin_task_<name>_<server>__<tool>,
// since Claude Code may spell the plugin's dash as an underscore), or,
// for each claude.ai connector mcp.connectors names,
// mcp__claude_ai_<connector>__<tool>. The same tool name served by
// another MCP server or connector does not count.
func (m *Manifest) MissingTools(available []string) []string {
	plugin := "task-" + m.Name
	prefixes := []string{
		"mcp__" + m.MCP.Server + "__",
		"mcp__plugin_" + plugin + "_" + m.MCP.Server + "__",
		"mcp__plugin_" + strings.ReplaceAll(plugin, "-", "_") + "_" + m.MCP.Server + "__",
	}
	for _, c := range m.MCP.Connectors {
		prefixes = append(prefixes, ConnectorToolPrefix(c))
	}

	var missing []string
	for _, tool := range m.MCP.Tools {
		found := slices.Contains(available, tool)
		for _, p := range prefixes {
			found = found || slices.Contains(available, p+tool)
		}
		if !found {
			missing = append(missing, tool)
		}
	}
	return missing
}
