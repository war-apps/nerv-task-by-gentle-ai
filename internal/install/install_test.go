package install_test

import (
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
