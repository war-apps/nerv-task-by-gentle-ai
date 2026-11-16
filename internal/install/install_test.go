package install_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner/runnertest"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/install"
)

func fakeCoreFS() fstest.MapFS {
	return fstest.MapFS{
		".claude-plugin/plugin.json": &fstest.MapFile{Data: []byte(`{"name":"task","version":"1.0.0"}`)},
		"commands/me.md":             &fstest.MapFile{Data: []byte("x")},
	}
}

func fakeExtensionsFS() fstest.MapFS {
	return fstest.MapFS{
		"task-teamwork/.claude-plugin/plugin.json": &fstest.MapFile{Data: []byte(`{"name":"task-teamwork","version":"1.0.0"}`)},
		"task-teamwork/provider.yaml":              &fstest.MapFile{Data: []byte("x")},
	}
}

func newDeps(t *testing.T) (install.Deps, string) {
	t.Helper()
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	marketplaceDir := filepath.Join(home, ".nerv-task", "marketplace")
	return install.Deps{
		Home:           home,
		SettingsPath:   settingsPath,
		MarketplaceDir: marketplaceDir,
		CoreFS:         fakeCoreFS(),
		ExtensionsFS:   fakeExtensionsFS(),
		Runner:         &runnertest.FakeRunner{},
		Now:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Stdout:         os.Stdout,
	}, settingsPath
}

func readSettings(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parsing settings: %v", err)
	}
	return m
}

func TestInstall_MaterializesAndRegistersCorePlugin(t *testing.T) {
	deps, settingsPath := newDeps(t)

	if _, err := install.Install(deps); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(deps.MarketplaceDir, "plugin", ".claude-plugin", "plugin.json")); err != nil {
		t.Errorf("core plugin was not materialized: %v", err)
	}

	m := readSettings(t, settingsPath)
	enabled, _ := m["enabledPlugins"].(map[string]any)
	if enabled == nil || enabled["task@nerv-task"] != true {
		t.Errorf("enabledPlugins = %v, want task@nerv-task enabled", enabled)
	}
	marketplaces, _ := m["extraKnownMarketplaces"].(map[string]any)
	if marketplaces == nil {
		t.Fatal("extraKnownMarketplaces missing")
	}
	if _, ok := marketplaces["nerv-task"]; !ok {
		t.Error("extraKnownMarketplaces does not register nerv-task")
	}
}

// TestInstall_MaterializedMarketplaceListsCorePluginAsTask guards against
// nerv-kit's own marketplace.json bug (fixed in v0.1.2): the core plugin
// must be listed under its own plugin.json name ("task"), matching the
// name half of CorePluginID ("task@nerv-task"), never the marketplace
// name ("nerv-task") the directory marketplace itself is registered
// under.
func TestInstall_MaterializedMarketplaceListsCorePluginAsTask(t *testing.T) {
	deps, _ := newDeps(t)

	if _, err := install.Install(deps); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(deps.MarketplaceDir, ".claude-plugin", "marketplace.json"))
	if err != nil {
		t.Fatalf("reading marketplace.json: %v", err)
	}
	var manifest struct {
		Plugins []struct {
			Name string `json:"name"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsing marketplace.json: %v", err)
	}
	if len(manifest.Plugins) == 0 {
		t.Fatal("marketplace.json lists no plugins")
	}
	if manifest.Plugins[0].Name != "task" {
		t.Errorf("marketplace.json core plugin name = %q, want %q", manifest.Plugins[0].Name, "task")
	}
}

func TestInstall_WithTeamwork_EnablesTeamworkPlugin(t *testing.T) {
	deps, settingsPath := newDeps(t)
	deps.WithTeamwork = true

	if _, err := install.Install(deps); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	m := readSettings(t, settingsPath)
	enabled := m["enabledPlugins"].(map[string]any)
	if enabled["task-teamwork@nerv-task"] != true {
		t.Errorf("enabledPlugins = %v, want task-teamwork@nerv-task enabled", enabled)
	}
}

func TestInstall_WithoutTeamwork_NeverEnablesTeamworkPlugin(t *testing.T) {
	deps, settingsPath := newDeps(t)

	if _, err := install.Install(deps); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	m := readSettings(t, settingsPath)
	enabled := m["enabledPlugins"].(map[string]any)
	if _, ok := enabled["task-teamwork@nerv-task"]; ok {
		t.Error("task-teamwork@nerv-task should not be enabled without --with-teamwork")
	}
}

func TestInstall_DisablesLegacyNervPlugins(t *testing.T) {
	deps, settingsPath := newDeps(t)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{"enabledPlugins":{"task@nerv":true,"task-teamwork@nerv":true,"release@nerv":true},"extraKnownMarketplaces":{"nerv":{"source":{"source":"directory","path":"/somewhere"}}}}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := install.Install(deps)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	m := readSettings(t, settingsPath)
	enabled := m["enabledPlugins"].(map[string]any)
	if _, ok := enabled["task@nerv"]; ok {
		t.Error("task@nerv should have been disabled")
	}
	if _, ok := enabled["task-teamwork@nerv"]; ok {
		t.Error("task-teamwork@nerv should have been disabled")
	}
	if enabled["release@nerv"] != true {
		t.Error("release@nerv must never be touched")
	}
	marketplaces := m["extraKnownMarketplaces"].(map[string]any)
	if _, ok := marketplaces["nerv"]; !ok {
		t.Error("the nerv marketplace must never be unregistered")
	}
	if len(result.DisabledLegacyPlugins) != 2 {
		t.Errorf("DisabledLegacyPlugins = %v, want 2 entries", result.DisabledLegacyPlugins)
	}
}

// TestInstall_FirstInstall_SkipsUninstallWhenPluginNotYetInstalled covers a
// real install.ps1 run: on a first install, task@nerv-task is not in
// installed_plugins.json yet, so "claude plugin uninstall task@nerv-task"
// fails with "Plugin ... not found in installed plugins" -- wording that
// does not contain "not installed" and that this package must not depend
// on. Install must check installed_plugins.json itself and skip the
// uninstall call entirely instead of treating that failure as fatal.
func TestInstall_FirstInstall_SkipsUninstallWhenPluginNotYetInstalled(t *testing.T) {
	deps, _ := newDeps(t)
	fake := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin uninstall task@nerv-task": {
				Stderr:   `Plugin "task@nerv-task" not found in installed plugins`,
				ExitCode: 1,
			},
		},
	}
	deps.Runner = fake

	if _, err := install.Install(deps); err != nil {
		t.Fatalf("Install() error = %v, want nil on a first install", err)
	}

	for _, c := range fake.Calls {
		if c.Name == "claude" && len(c.Args) >= 2 && c.Args[0] == "plugin" && c.Args[1] == "uninstall" {
			t.Errorf("Install() called claude plugin uninstall on a first install, want it skipped: %v", c.Args)
		}
	}
}

// TestInstall_WhenAlreadyInstalled_UninstallsBeforeReinstalling keeps the
// refresh behaviour for a second install: when installed_plugins.json
// already records the plugin, Install must still uninstall it before
// reinstalling, so an upgraded bundle is actually picked up.
func TestInstall_WhenAlreadyInstalled_UninstallsBeforeReinstalling(t *testing.T) {
	deps, _ := newDeps(t)
	writeInstalledPlugins(t, deps.Home, map[string]string{"task@nerv-task": deps.MarketplaceDir})
	fake := &runnertest.FakeRunner{}
	deps.Runner = fake

	if _, err := install.Install(deps); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	uninstallIdx, installIdx := -1, -1
	for i, c := range fake.Calls {
		if c.Name == "claude" && len(c.Args) >= 3 && c.Args[0] == "plugin" && c.Args[1] == "uninstall" && c.Args[2] == "task@nerv-task" {
			uninstallIdx = i
		}
		if c.Name == "claude" && len(c.Args) >= 3 && c.Args[0] == "plugin" && c.Args[1] == "install" && c.Args[2] == "task@nerv-task" {
			installIdx = i
		}
	}
	if uninstallIdx == -1 {
		t.Error("Install() never uninstalled the already-installed plugin before reinstalling it")
	}
	if installIdx == -1 || installIdx < uninstallIdx {
		t.Errorf("Install() must reinstall after uninstalling (uninstall at %d, install at %d)", uninstallIdx, installIdx)
	}
}

// TestInstall_UninstallFailsForOtherReason_PropagatesError makes sure the
// installed_plugins.json check only tolerates the "not yet installed"
// case: any other uninstall failure on a plugin that IS recorded as
// installed must still fail Install.
func TestInstall_UninstallFailsForOtherReason_PropagatesError(t *testing.T) {
	deps, _ := newDeps(t)
	writeInstalledPlugins(t, deps.Home, map[string]string{"task@nerv-task": deps.MarketplaceDir})
	fake := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin uninstall task@nerv-task": {Stderr: "network error talking to the plugin registry", ExitCode: 1},
		},
	}
	deps.Runner = fake

	if _, err := install.Install(deps); err == nil {
		t.Fatal("Install() error = nil, want an error for an unrelated uninstall failure")
	}
}

// TestInstall_LegacyTeamworkEnabled_AutoInstallsTeamworkPlugin covers the
// silent loss of the Teamwork provider: when task-teamwork@nerv was
// enabled before install, and --with-teamwork was not passed (no terminal
// to offer it either), task-teamwork@nerv-task must be installed anyway
// so /task:* keeps its Teamwork provider.
func TestInstall_LegacyTeamworkEnabled_AutoInstallsTeamworkPlugin(t *testing.T) {
	deps, settingsPath := newDeps(t)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{"enabledPlugins":{"task@nerv":true,"task-teamwork@nerv":true}}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	deps.Stdout = &stdout

	result, err := install.Install(deps)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if !result.TeamworkAutoEnabled {
		t.Error("Result.TeamworkAutoEnabled = false, want true")
	}
	m := readSettings(t, settingsPath)
	enabled := m["enabledPlugins"].(map[string]any)
	if enabled["task-teamwork@nerv-task"] != true {
		t.Errorf("enabledPlugins = %v, want task-teamwork@nerv-task enabled automatically", enabled)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("task-teamwork@nerv-task")) {
		t.Errorf("stdout = %q, want it to say task-teamwork@nerv-task was installed automatically", stdout.String())
	}
}

// TestInstall_OnInstallFailure_LeavesLegacyPluginsEnabled is the rollback
// safety net: if installing the new plugins through the real CLI fails,
// settings.json must stay exactly as it was, so the user is never left
// without a working /task:* (the old nerv plugins stay enabled).
func TestInstall_OnInstallFailure_LeavesLegacyPluginsEnabled(t *testing.T) {
	deps, settingsPath := newDeps(t)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{"enabledPlugins":{"task@nerv":true,"release@nerv":true},"extraKnownMarketplaces":{"nerv":{"source":{"source":"directory","path":"/somewhere"}}}}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin install task@nerv-task": {Stderr: `Plugin "task" not found in marketplace "nerv-task"`, ExitCode: 1},
		},
	}
	deps.Runner = fake

	result, err := install.Install(deps)
	if err == nil {
		t.Fatal("Install() error = nil, want an error when the CLI install fails")
	}
	if len(result.DisabledLegacyPlugins) != 0 {
		t.Errorf("DisabledLegacyPlugins = %v, want none on a failed install", result.DisabledLegacyPlugins)
	}

	after, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}
	if string(after) != existing {
		t.Errorf("settings.json changed on a failed install:\nbefore: %s\nafter:  %s", existing, after)
	}
}

// writeInstalledPlugins writes a minimal installed_plugins.json under
// home/.claude/plugins recording each given plugin id as installed at
// installPath.
func writeInstalledPlugins(t *testing.T, home string, plugins map[string]string) {
	t.Helper()
	type entry struct {
		InstallPath string `json:"installPath"`
		Version     string `json:"version"`
	}
	doc := struct {
		Plugins map[string][]entry `json:"plugins"`
	}{Plugins: map[string][]entry{}}
	for id, path := range plugins {
		doc.Plugins[id] = []entry{{InstallPath: path, Version: "1.0.0"}}
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "installed_plugins.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstall_RefreshesPluginCache(t *testing.T) {
	deps, _ := newDeps(t)
	fake := &runnertest.FakeRunner{}
	deps.Runner = fake

	if _, err := install.Install(deps); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	foundAdd := false
	foundInstall := false
	for _, c := range fake.Calls {
		if c.Name == "claude" && len(c.Args) >= 3 && c.Args[0] == "plugin" && c.Args[1] == "marketplace" && c.Args[2] == "add" {
			foundAdd = true
		}
		if c.Name == "claude" && len(c.Args) >= 2 && c.Args[0] == "plugin" && c.Args[1] == "install" {
			foundInstall = true
		}
	}
	if !foundAdd {
		t.Error("Install() never ran claude plugin marketplace add")
	}
	if !foundInstall {
		t.Error("Install() never ran claude plugin install")
	}
}
