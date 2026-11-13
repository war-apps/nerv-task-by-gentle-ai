package upgrade_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner/runnertest"
	"github.com/war-apps/nerv-kit-by-gentle-ai/selfupgrade"
	"github.com/war-apps/nerv-kit-by-gentle-ai/selfupgrade/selfupgradetest"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/upgrade"
)

func newDeps(t *testing.T, srv *selfupgradetest.Server, current string) upgrade.Deps {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "nerv-task")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return upgrade.Deps{
		Client: selfupgrade.Client{
			HTTP:         srv.Client(),
			APIBase:      srv.APIBase(),
			DownloadBase: srv.DownloadBase(),
			BinaryName:   "nerv-task",
		},
		Channel:        selfupgrade.Stable,
		CurrentVersion: current,
		GOOS:           "linux",
		GOARCH:         "amd64",
		BinaryName:     "nerv-task",
		Exe:            exe,
		Runner:         &runnertest.FakeRunner{},
	}
}

func TestUpgrade_AlreadyUpToDate_DoesNotReplace(t *testing.T) {
	srv := selfupgradetest.New(t, "nerv-task", selfupgradetest.Release{Tag: "v1.0.0"})
	deps := newDeps(t, srv, "v1.0.0")

	result, err := upgrade.Upgrade(context.Background(), deps)
	if err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if !result.UpToDate {
		t.Error("UpToDate = false, want true")
	}

	data, err := os.ReadFile(deps.Exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old binary" {
		t.Error("the executable was replaced, want it untouched when already up to date")
	}
}

func TestUpgrade_NewerRelease_ReplacesAndReinstalls(t *testing.T) {
	srv := selfupgradetest.New(t, "nerv-task", selfupgradetest.Release{Tag: "v1.1.0"})
	deps := newDeps(t, srv, "v1.0.0")

	result, err := upgrade.Upgrade(context.Background(), deps)
	if err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if !result.Upgraded {
		t.Error("Upgraded = false, want true")
	}
	if result.To != "v1.1.0" {
		t.Errorf("To = %q, want v1.1.0", result.To)
	}

	data, err := os.ReadFile(deps.Exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "old binary" {
		t.Error("the executable was never replaced")
	}

	fake := deps.Runner.(*runnertest.FakeRunner)
	found := false
	for _, c := range fake.Calls {
		if c.Name == deps.Exe && len(c.Args) == 1 && c.Args[0] == "install" {
			found = true
		}
	}
	if !found {
		t.Errorf("Upgrade() never re-ran install; calls = %v", fake.Calls)
	}
}

func TestUpgrade_UnsupportedPlatform_Refuses(t *testing.T) {
	srv := selfupgradetest.New(t, "nerv-task", selfupgradetest.Release{Tag: "v1.1.0"})
	deps := newDeps(t, srv, "v1.0.0")
	deps.GOOS = "plan9"

	_, err := upgrade.Upgrade(context.Background(), deps)
	if err == nil {
		t.Fatal("Upgrade() error = nil, want a refusal for an unsupported platform")
	}
}

func TestUpgrade_AlphaChannel_ResolvesPrerelease(t *testing.T) {
	srv := selfupgradetest.New(t, "nerv-task",
		selfupgradetest.Release{Tag: "v1.1.0-alpha.1", Prerelease: true},
		selfupgradetest.Release{Tag: "v1.0.0"},
	)
	deps := newDeps(t, srv, "v1.0.0")
	deps.Channel = selfupgrade.Alpha

	result, err := upgrade.Upgrade(context.Background(), deps)
	if err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if result.To != "v1.1.0-alpha.1" {
		t.Errorf("To = %q, want v1.1.0-alpha.1", result.To)
	}
}
