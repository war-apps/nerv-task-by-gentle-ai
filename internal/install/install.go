// Package install materializes and registers the nerv-task plugin
// marketplace in Claude Code, mirroring nerv's own install flow (kit
// claudeplugin) but scoped to nerv-task's own marketplace and plugins
// only.
package install

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

// legacyTeamworkPluginID is nerv's own task-teamwork plugin id. When it
// was enabled before install, task-teamwork@nerv-task is installed
// automatically (see Install), so the Teamwork provider is never silently
// lost just because --with-teamwork was not passed and there was no
// terminal to offer it.
const legacyTeamworkPluginID = "task-teamwork@nerv"

// LegacyPluginIDs are the plugin ids nerv itself ships for task tracking.
// Installing nerv-task disables them (never removes the nerv
// marketplace, and never touches any other nerv plugin), so /task:*
// commands are never duplicated between the two marketplaces.
var LegacyPluginIDs = []string{"task@nerv", legacyTeamworkPluginID}

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
	// TeamworkAutoEnabled reports whether task-teamwork@nerv-task was
	// installed automatically because task-teamwork@nerv was enabled
	// before install, even though the caller did not ask for
	// --with-teamwork.
	TeamworkAutoEnabled bool
}

// Install materializes the nerv-task plugin bundle to deps.MarketplaceDir,
// registers it with the real claude CLI, installs the plugins the caller
// selected (plus task-teamwork@nerv-task when nerv's own
// task-teamwork@nerv was enabled), and only then disables nerv's own task
// plugins and writes Claude Code's settings.json. Keeping the CLI install
// before the settings write means that on any failure the old nerv
// plugins -- when they were enabled -- are left exactly as they were, so
// /task:* is never left without a working provider.
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

	withTeamwork := deps.WithTeamwork
	if !withTeamwork && settings.PluginEnabled(legacyTeamworkPluginID) {
		withTeamwork = true
		result.TeamworkAutoEnabled = true
		if deps.Stdout != nil {
			fmt.Fprintf(deps.Stdout, "%s was enabled; installing %s automatically so the Teamwork provider is not lost.\n", legacyTeamworkPluginID, TeamworkPluginID)
		}
	}

	ids := []string{CorePluginID}
	if withTeamwork {
		ids = append(ids, TeamworkPluginID)
	}
	// Every nerv-task plugin the caller already had enabled is refreshed
	// too, regardless of --with-teamwork or a terminal offering it: this
	// is what keeps `nerv-task upgrade` (which re-runs install with no
	// flags and no terminal) from silently leaving an enabled plugin
	// behind at its old installed version. Never enables a plugin the
	// caller did not already have.
	for _, id := range settings.MarketplacePlugins(MarketplaceName) {
		if settings.PluginEnabled(id) && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}

	if err := refreshCache(deps, ids); err != nil {
		return result, err
	}

	settings.RegisterMarketplace(MarketplaceName, deps.MarketplaceDir)
	for _, id := range ids {
		settings.EnablePlugin(id)
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

// refreshCache registers deps.MarketplaceDir with the real claude CLI and
// installs every id in ids, uninstalling first when the cache already
// carries an older copy (so an upgraded bundle is actually picked up).
//
// On a first install nothing is installed yet, and asking the CLI to
// uninstall a plugin it never installed fails ("Plugin ... not found in
// installed plugins", or any other wording Claude Code chooses) in a way
// this package must not depend on to tell apart from a real failure.
// installed_plugins.json is checked first instead (kit
// claudeplugin.InstalledPlugin), and the uninstall call is skipped
// entirely when the plugin is not there -- any other uninstall failure,
// for a plugin that IS recorded as installed, still fails Install.
func refreshCache(deps Deps, ids []string) error {
	cli := claudeplugin.PluginCLI{Runner: deps.Runner}
	ctx := context.Background()

	if _, err := cli.AddMarketplace(ctx, deps.MarketplaceDir); err != nil {
		return err
	}

	installedPath := filepath.Join(deps.Home, ".claude", "plugins", "installed_plugins.json")

	for _, id := range ids {
		_, installed, err := claudeplugin.InstalledPlugin(installedPath, id)
		if err != nil {
			return err
		}
		if installed {
			if _, err := cli.Uninstall(ctx, id); err != nil {
				return err
			}
		}
		if _, err := cli.Install(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
