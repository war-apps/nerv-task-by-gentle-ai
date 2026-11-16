// Package configure implements `nerv-task configure`: printing and
// editing the global task.yaml, and the built-in (or opportunistically
// read) git settings the worktree offer uses.
package configure

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

// Deps bundles what Print and Set need: the global task.yaml path, the
// discovered provider manifests (for listing and --set validation), and
// the repository's git settings.
type Deps struct {
	GlobalPath string
	Providers  []*provider.Manifest
	Git        GitSettings
	Now        func() time.Time
	Stdin      io.Reader
	Stdout     io.Writer
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d Deps) findProvider(name string) *provider.Manifest {
	for _, m := range d.Providers {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// ProviderSummary is one provider's config schema, as --print reports it.
type ProviderSummary struct {
	Name          string               `json:"name"`
	TaskRefPrefix string               `json:"task_ref_prefix"`
	Config        []provider.ConfigKey `json:"config"`
}

// PrintOutput is the result of `nerv-task configure --print`.
type PrintOutput struct {
	ConfigPath string              `json:"config_path"`
	Values     config.GlobalConfig `json:"values"`
	Defaults   config.RepoConfig   `json:"defaults"`
	Providers  []ProviderSummary   `json:"providers"`
	Git        GitSettings         `json:"git"`
}

// Print reads the global task.yaml and reports its values, its defaults
// seed, every known provider's config schema, and the resolved git
// settings.
func Print(deps Deps) (PrintOutput, error) {
	values, _, err := config.LoadGlobal(deps.GlobalPath)
	if err != nil {
		return PrintOutput{}, err
	}
	out := PrintOutput{
		ConfigPath: deps.GlobalPath,
		Values:     values,
		Git:        deps.Git,
	}
	if out.Git == (GitSettings{}) {
		out.Git = DefaultGit()
	}
	if values.Defaults != nil {
		out.Defaults = *values.Defaults
	}
	for _, m := range deps.Providers {
		out.Providers = append(out.Providers, ProviderSummary{
			Name:          m.Name,
			TaskRefPrefix: m.TaskRefPrefix,
			Config:        m.Config,
		})
	}
	return out, nil
}

// SetResult reports one `--set key=value` mutation.
type SetResult struct {
	Key string
	Old string
	New string
}

// Set parses "key=value" (either a global scalar key, or
// "<provider>.<key>" for a provider's scope: user config key),
// validates it against the known schema, and persists it to the global
// task.yaml.
func Set(deps Deps, kv string) (SetResult, error) {
	key, value, ok := strings.Cut(kv, "=")
	if !ok {
		return SetResult{}, fmt.Errorf("invalid --set %q: want key=value", kv)
	}

	cfg, _, err := config.LoadGlobal(deps.GlobalPath)
	if err != nil {
		return SetResult{}, err
	}

	var result SetResult
	result.Key = key

	switch key {
	case "timer_store":
		result.Old = cfg.TimerStore
		cfg.TimerStore = value
		result.New = value
	case "stale_after_hours", "rounding_minutes":
		n, err := strconv.Atoi(value)
		if err != nil {
			return SetResult{}, fmt.Errorf("%s: %q is not a valid int", key, value)
		}
		if key == "stale_after_hours" {
			if cfg.StaleAfterHours != nil {
				result.Old = strconv.Itoa(*cfg.StaleAfterHours)
			}
			cfg.StaleAfterHours = &n
		} else {
			if cfg.RoundingMinutes != nil {
				result.Old = strconv.Itoa(*cfg.RoundingMinutes)
			}
			cfg.RoundingMinutes = &n
		}
		result.New = value
	default:
		providerName, configKey, ok := strings.Cut(key, ".")
		if !ok {
			return SetResult{}, fmt.Errorf("unknown key %q", key)
		}
		m := deps.findProvider(providerName)
		if m == nil {
			return SetResult{}, fmt.Errorf("provider %q is not installed", providerName)
		}
		ck, ok := findConfigKey(m.Config, configKey)
		if !ok {
			return SetResult{}, fmt.Errorf("provider %s has no config key %q", providerName, configKey)
		}
		if ck.Scope != "user" {
			return SetResult{}, fmt.Errorf("%s.%s is a scope: repo key; it belongs in the repository's task.yaml (nerv-task init or /task:setup), not the global configuration", providerName, configKey)
		}
		typedValue, err := typedConfigValue(ck.Type, value)
		if err != nil {
			return SetResult{}, fmt.Errorf("%s.%s: %w", providerName, configKey, err)
		}
		if cfg.Providers == nil {
			cfg.Providers = map[string]map[string]any{}
		}
		if cfg.Providers[providerName] == nil {
			cfg.Providers[providerName] = map[string]any{}
		}
		if old, ok := cfg.Providers[providerName][configKey]; ok {
			result.Old = fmt.Sprint(old)
		}
		cfg.Providers[providerName][configKey] = typedValue
		result.New = fmt.Sprint(typedValue)
	}

	if _, _, err := config.SaveGlobal(deps.GlobalPath, cfg, deps.now()); err != nil {
		return SetResult{}, err
	}
	return result, nil
}

func findConfigKey(keys []provider.ConfigKey, key string) (provider.ConfigKey, bool) {
	for _, k := range keys {
		if k.Key == key {
			return k, true
		}
	}
	return provider.ConfigKey{}, false
}

func typedConfigValue(typ, value string) (any, error) {
	switch typ {
	case "int":
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid int", value)
		}
		return n, nil
	case "bool":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid bool", value)
		}
		return b, nil
	default:
		return value, nil
	}
}
