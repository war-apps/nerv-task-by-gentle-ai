// Package config is nerv-task's two configuration files (kit
// aiworkflow): the repository's own task.yaml
// (<repo>/.ai-workflow/task.yaml) and the global task.yaml
// (~/.config/ai-workflow/task.yaml on Linux, %AppData%\ai-workflow\task.yaml
// on Windows). See plugin/task/skills/task/SKILL.md §1 for the full key
// reference.
package config

import (
	"time"

	"github.com/war-apps/nerv-kit-by-gentle-ai/aiworkflow"
)

// Tool is the name nerv-task's files are resolved under.
const Tool = "task"

// DefaultTimerStore and DefaultStaleAfterHours are the global file's
// built-in defaults (plugin/task/skills/task/SKILL.md §1).
const (
	DefaultTimerStore      = "~/.claude/work/timers.json"
	DefaultStaleAfterHours = 12
)

// RepoConfig is the repository's own task.yaml.
type RepoConfig struct {
	Provider        string                    `yaml:"provider,omitempty"`
	AskWhenMissing  *bool                     `yaml:"ask_when_missing,omitempty"`
	SubtasksPerWave *bool                     `yaml:"subtasks_per_wave,omitempty"`
	Events          map[string]string         `yaml:"events,omitempty"`
	Sources         []map[string]any          `yaml:"sources,omitempty"`
	Providers       map[string]map[string]any `yaml:"providers,omitempty"`
}

// GlobalConfig is the global task.yaml: the machine-scoped and
// scope: user keys, plus the defaults seed for a new repository file.
type GlobalConfig struct {
	TimerStore      string                    `yaml:"timer_store,omitempty"`
	StaleAfterHours *int                      `yaml:"stale_after_hours,omitempty"`
	RoundingMinutes *int                      `yaml:"rounding_minutes,omitempty"`
	Providers       map[string]map[string]any `yaml:"providers,omitempty"`
	Defaults        *RepoConfig               `yaml:"defaults,omitempty"`
}

// ProviderValue returns the repository file's providers.<name>.<key>
// value, or nil when unset.
func (c RepoConfig) ProviderValue(name, key string) any {
	p, ok := c.Providers[name]
	if !ok {
		return nil
	}
	return p[key]
}

// ProviderValue returns the global file's providers.<name>.<key> value,
// or nil when unset.
func (c GlobalConfig) ProviderValue(name, key string) any {
	p, ok := c.Providers[name]
	if !ok {
		return nil
	}
	return p[key]
}

// UserPath is the global task.yaml path for home on goos.
func UserPath(home, goos string) string {
	return aiworkflow.UserConfigPath(home, goos, Tool)
}

// RepoPath is the repository's own task.yaml path.
func RepoPath(root string) string {
	return aiworkflow.ProjectConfigPath(root, Tool)
}

// LoadGlobal reads the global task.yaml at path. A missing file is not
// an error: found is false and the zero GlobalConfig is returned.
func LoadGlobal(path string) (GlobalConfig, bool, error) {
	var cfg GlobalConfig
	found, err := aiworkflow.Load(path, &cfg)
	return cfg, found, err
}

// SaveGlobal writes cfg to the global task.yaml at path, atomically,
// with a timestamped backup of any existing file.
func SaveGlobal(path string, cfg GlobalConfig, now time.Time) (bool, string, error) {
	return aiworkflow.Save(path, cfg, aiworkflow.SaveOptions{Now: now, BackupSuffix: "bak-configure-"})
}

// LoadRepo reads the repository's own task.yaml at path. A missing file
// is not an error: found is false and the zero RepoConfig is returned.
func LoadRepo(path string) (RepoConfig, bool, error) {
	var cfg RepoConfig
	found, err := aiworkflow.Load(path, &cfg)
	return cfg, found, err
}

// SaveRepo writes cfg to the repository's own task.yaml at path,
// atomically, with a timestamped backup of any existing file.
func SaveRepo(path string, cfg RepoConfig, now time.Time) (bool, string, error) {
	return aiworkflow.Save(path, cfg, aiworkflow.SaveOptions{Now: now, BackupSuffix: "bak-init-"})
}
