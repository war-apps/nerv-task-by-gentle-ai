// Package install materializes and registers the nerv-task plugin
// marketplace in Claude Code, mirroring nerv's own install flow (kit
// claudeplugin) but scoped to nerv-task's own marketplace and plugins
// only.
package install

import (
	"context"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/claudeplugin"
	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"
)

// MarketplaceName is the directory marketplace nerv-task registers
// itself under.
const MarketplaceName = "nerv-task"

// CorePluginID and TeamworkPluginID are this marketplace's plugin ids as
// Claude Code's settings.json enabledPlugins keys them.
const (
	CorePluginID     = "task@" + MarketplaceName
	TeamworkPluginID = "task-teamwork@" + MarketplaceName
)

// LegacyPluginIDs are the plugin ids nerv itself ships for task tracking.
// Installing nerv-task disables them (never removes the nerv
// marketplace, and never touches any other nerv plugin), so /task:*
// commands are never duplicated between the two marketplaces.
var LegacyPluginIDs = []string{"task@nerv", "task-teamwork@nerv"}

// OwnerName is the marketplace owner nerv-task registers as.
const OwnerName = "Walter Rodriguez"

// Deps bundles every external effect Install needs, so it can be tested
// against a temporary HOME, a temporary settings.json and a fake runner,
// with no real network and no real claude binary.
type Deps struct {
	Home           string
	SettingsPath   string
	MarketplaceDir string
	CoreFS         fs.FS
	ExtensionsFS   fs.FS
	Runner         runner.Runner
	Now            time.Time
	Stdout         io.Writer
	// WithTeamwork enables the task-teamwork provider plugin in
	// addition to the core task plugin.
	WithTeamwork bool
}

// Result reports what Install did.
type Result struct {
	Materialized          claudeplugin.Result
	DisabledLegacyPlugins []string
	SettingsWritten       bool
}

// Install materializes the nerv-task plugin bundle to deps.MarketplaceDir,
// registers it (and the plugins the caller selected) in Claude Code's
// settings.json, disables nerv's own task plugins when they were
// enabled, and refreshes the plugin cache through the claude CLI.
func Install(deps Deps) (Result, error) {
	var result Result

	materialized, err := claudeplugin.MaterializeBundle(deps.CoreFS, deps.ExtensionsFS, deps.MarketplaceDir, claudeplugin.Marketplace{
		Name:      MarketplaceName,
		OwnerName: OwnerName,
	})
	if err != nil {
		return result, err
	}
	result.Materialized = materialized

	previous, settings, err := loadOrCreateSettings(deps.SettingsPath)
	if err != nil {
		return result, err
	}

	settings.RegisterMarketplace(MarketplaceName, deps.MarketplaceDir)
	settings.EnablePlugin(CorePluginID)
	if deps.WithTeamwork {
		settings.EnablePlugin(TeamworkPluginID)
	}

	for _, id := range LegacyPluginIDs {
		if settings.PluginEnabled(id) {
			settings.RemovePlugin(id)
			result.DisabledLegacyPlugins = append(result.DisabledLegacyPlugins, id)
		}
	}

	written, _, err := claudeplugin.SaveSettings(deps.SettingsPath, settings, previous, deps.Now, "bak-install-")
	if err != nil {
		return result, err
	}
	result.SettingsWritten = written

	if err := refreshCache(deps); err != nil {
		return result, err
	}

	return result, nil
}

func loadOrCreateSettings(path string) ([]byte, *claudeplugin.Settings, error) {
	previous, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, claudeplugin.NewSettings(), nil
		}
		return nil, nil, err
	}
	settings, err := claudeplugin.LoadSettings(path)
	if err != nil {
		return nil, nil, err
	}
	return previous, settings, nil
}

func refreshCache(deps Deps) error {
	cli := claudeplugin.PluginCLI{Runner: deps.Runner}
	ctx := context.Background()

	if _, err := cli.AddMarketplace(ctx, deps.MarketplaceDir); err != nil {
		return err
	}

	ids := []string{CorePluginID}
	if deps.WithTeamwork {
		ids = append(ids, TeamworkPluginID)
	}
	for _, id := range ids {
		if _, err := cli.Uninstall(ctx, id); err != nil {
			return err
		}
		if _, err := cli.Install(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
