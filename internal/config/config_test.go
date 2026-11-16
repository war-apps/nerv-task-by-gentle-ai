package config_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
)

func TestUserPath_Linux(t *testing.T) {
	got := config.UserPath("/home/u", "linux")
	want := filepath.Join("/home/u", ".config", "ai-workflow", "task.yaml")
	if got != want {
		t.Errorf("UserPath() = %q, want %q", got, want)
	}
}

func TestRepoPath(t *testing.T) {
	got := config.RepoPath(filepath.FromSlash("/repo"))
	want := filepath.Join("/repo", ".ai-workflow", "task.yaml")
	if got != want {
		t.Errorf("RepoPath() = %q, want %q", got, want)
	}
}

func TestLoadGlobal_MissingFileIsNotFoundNoError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.yaml")
	cfg, found, err := config.LoadGlobal(path)
	if err != nil {
		t.Fatalf("LoadGlobal() error = %v", err)
	}
	if found {
		t.Error("LoadGlobal() found = true, want false for a missing file")
	}
	if cfg.TimerStore != "" {
		t.Errorf("LoadGlobal() cfg = %+v, want zero value", cfg)
	}
}

func TestSaveGlobal_ThenLoadGlobal_RoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.yaml")
	cfg := config.GlobalConfig{
		TimerStore: "~/.claude/work/timers.json",
		Providers: map[string]map[string]any{
			"teamwork": {"assignee_id": 100},
		},
	}
	written, _, err := config.SaveGlobal(path, cfg, time.Now())
	if err != nil {
		t.Fatalf("SaveGlobal() error = %v", err)
	}
	if !written {
		t.Error("SaveGlobal() written = false, want true for a new file")
	}

	got, found, err := config.LoadGlobal(path)
	if err != nil {
		t.Fatalf("LoadGlobal() error = %v", err)
	}
	if !found {
		t.Fatal("LoadGlobal() found = false, want true")
	}
	if got.TimerStore != cfg.TimerStore {
		t.Errorf("TimerStore = %q, want %q", got.TimerStore, cfg.TimerStore)
	}
	if got.Providers["teamwork"]["assignee_id"] != 100 {
		t.Errorf("Providers[teamwork][assignee_id] = %v, want 100", got.Providers["teamwork"]["assignee_id"])
	}
}

func TestSaveRepo_ThenLoadRepo_RoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.yaml")
	trueVal := true
	cfg := config.RepoConfig{
		Provider:       "teamwork",
		AskWhenMissing: &trueVal,
		Providers: map[string]map[string]any{
			"teamwork": {"project_id": 1},
		},
	}
	if _, _, err := config.SaveRepo(path, cfg, time.Now()); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	got, found, err := config.LoadRepo(path)
	if err != nil {
		t.Fatalf("LoadRepo() error = %v", err)
	}
	if !found {
		t.Fatal("LoadRepo() found = false, want true")
	}
	if got.Provider != "teamwork" {
		t.Errorf("Provider = %q, want teamwork", got.Provider)
	}
	if got.AskWhenMissing == nil || !*got.AskWhenMissing {
		t.Errorf("AskWhenMissing = %v, want true", got.AskWhenMissing)
	}
}

func TestRepoConfig_ProviderValue_Defaulting(t *testing.T) {
	cfg := config.RepoConfig{Providers: map[string]map[string]any{
		"teamwork": {"project_id": 1},
	}}
	if got := cfg.ProviderValue("teamwork", "project_id"); got != 1 {
		t.Errorf("ProviderValue() = %v, want 1", got)
	}
	if got := cfg.ProviderValue("teamwork", "missing"); got != nil {
		t.Errorf("ProviderValue() = %v, want nil", got)
	}
	if got := cfg.ProviderValue("none", "x"); got != nil {
		t.Errorf("ProviderValue() = %v, want nil", got)
	}
}
