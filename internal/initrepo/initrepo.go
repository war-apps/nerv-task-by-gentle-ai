// Package initrepo implements `nerv-task init`: writing this
// repository's own task.yaml, seeded from the global file's defaults
// block, or imported 1:1 from nerv's own .ai-workflow/config.yaml
// tasks: block.
package initrepo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

// Deps bundles what Init needs.
type Deps struct {
	RepoRoot   string
	GlobalPath string
	Providers  []*provider.Manifest
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

// Options are nerv-task init's own flags.
type Options struct {
	// Provider, when set, overrides the seeded provider.
	Provider string
	// None skips writing the repository file entirely: task tracking
	// stays off in this repository.
	None bool
	// Import copies nerv's own .ai-workflow/config.yaml tasks: block
	// into the new repository file, dropping the tasks. prefix.
	Import bool
}

// Result reports what Init did.
type Result struct {
	Skipped            bool
	Path               string
	ImportedFrom       string
	SeededFromDefaults bool
	// SkippedPersonalKeys lists, as ready-to-print messages, every
	// scope: user key the imported tasks: block carried, that was
	// deliberately never written anywhere. A repository's own
	// .ai-workflow/config.yaml is untrusted input (it travels with a
	// clone); writing its personal keys into the shared global file
	// would poison every other repository from one repo's committed
	// file. Values are never echoed, only key names.
	SkippedPersonalKeys []string
	// InvalidKeys lists, as ready-to-print messages, every imported
	// scope: repo key whose value did not match its manifest's
	// declared type. Such a key is skipped, never written.
	InvalidKeys []string
}

// Init writes this repository's own task.yaml.
func Init(deps Deps, opts Options) (Result, error) {
	if opts.None {
		return Result{Skipped: true}, nil
	}

	repo := config.RepoConfig{}

	global, _, err := config.LoadGlobal(deps.GlobalPath)
	if err != nil {
		return Result{}, err
	}
	var result Result
	if global.Defaults != nil {
		repo = cloneRepoConfig(*global.Defaults)
		result.SeededFromDefaults = true
	}

	if opts.Import {
		nervConfigPath := filepath.Join(deps.RepoRoot, ".ai-workflow", "config.yaml")
		tasksBlock, found, err := readNervTasksBlock(nervConfigPath)
		if err != nil {
			return Result{}, err
		}
		if found {
			skipped, invalid := importTasksBlock(deps, &repo, tasksBlock)
			result.ImportedFrom = nervConfigPath
			result.SkippedPersonalKeys = skipped
			result.InvalidKeys = invalid
		}
	}

	if opts.Provider != "" {
		repo.Provider = opts.Provider
	}

	path := config.RepoPath(deps.RepoRoot)
	if _, _, err := config.SaveRepo(path, repo, deps.now()); err != nil {
		return Result{}, err
	}
	result.Path = path
	return result, nil
}

func cloneRepoConfig(src config.RepoConfig) config.RepoConfig {
	dst := src
	dst.Providers = nil
	for name, keys := range src.Providers {
		if dst.Providers == nil {
			dst.Providers = map[string]map[string]any{}
		}
		copied := map[string]any{}
		for k, v := range keys {
			copied[k] = v
		}
		dst.Providers[name] = copied
	}
	return dst
}

// readNervTasksBlock reads nerv's own .ai-workflow/config.yaml and
// returns its tasks: block as a generic map. A missing file, or one
// with no tasks: key, is not an error: found is false.
func readNervTasksBlock(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, false, fmt.Errorf("%s: %w", path, err)
	}
	tasks, ok := doc["tasks"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	return tasks, true, nil
}

// personalTopLevelKeys are the machine-scoped keys nerv's own tasks:
// block may carry at its top level. They are personal exactly like a
// scope: user provider key, and a repository's own file is untrusted
// input, so they are never imported anywhere -- only reported as
// skipped.
var personalTopLevelKeys = []string{"timer_store", "stale_after_hours", "rounding_minutes"}

// importTasksBlock copies tasks (nerv's tasks: block) into repo,
// dropping the tasks. prefix 1:1. A repository's own
// .ai-workflow/config.yaml travels with a clone and is untrusted input:
// only scope: repo provider keys (plus the always-repo enabled key) are
// imported into the new repository file. A scope: user provider key, or
// a personal top-level key, is never written anywhere -- not to the
// repository file, and never to the shared global file -- only
// reported in the returned skipped list. A scope: repo key whose value
// does not match its manifest's declared type is skipped too, reported
// in the returned invalid list.
func importTasksBlock(deps Deps, repo *config.RepoConfig, tasks map[string]any) (skipped, invalid []string) {
	if v, ok := tasks["provider"].(string); ok {
		repo.Provider = v
	}
	if v, ok := tasks["ask_when_missing"].(bool); ok {
		repo.AskWhenMissing = &v
	}
	if v, ok := tasks["subtasks_per_wave"].(bool); ok {
		repo.SubtasksPerWave = &v
	}
	if v, ok := tasks["events"].(map[string]any); ok {
		repo.Events = map[string]string{}
		for k, ev := range v {
			if s, ok := ev.(string); ok {
				repo.Events[k] = s
			}
		}
	}
	if v, ok := tasks["sources"].([]any); ok {
		for _, s := range v {
			if m, ok := s.(map[string]any); ok {
				repo.Sources = append(repo.Sources, m)
			}
		}
	}

	for _, key := range personalTopLevelKeys {
		if _, ok := tasks[key]; ok {
			skipped = append(skipped, fmt.Sprintf(
				"%s is personal; set it with `nerv-task configure --set %s=<value>`", key, key))
		}
	}

	providers, ok := tasks["providers"].(map[string]any)
	if !ok {
		return skipped, invalid
	}
	for _, name := range sortedKeys(providers) {
		keys, ok := providers[name].(map[string]any)
		if !ok {
			continue
		}
		m := deps.findProvider(name)
		for _, key := range sortedKeys(keys) {
			value := keys[key]
			qualified := name + "." + key

			scope := "repo"
			var declaredType string
			if key != "enabled" && m != nil {
				if ck, ok := findConfigKey(m.Config, key); ok {
					scope = ck.Scope
					declaredType = ck.Type
				}
			}

			if scope == "user" {
				skipped = append(skipped, fmt.Sprintf(
					"%s is personal; set it with `nerv-task configure --set %s=<value>`", qualified, qualified))
				continue
			}
			if declaredType != "" && !typeMatches(declaredType, value) {
				invalid = append(invalid, fmt.Sprintf(
					"%s: value does not match its declared type %s; skipped", qualified, declaredType))
				continue
			}

			if repo.Providers == nil {
				repo.Providers = map[string]map[string]any{}
			}
			if repo.Providers[name] == nil {
				repo.Providers[name] = map[string]any{}
			}
			repo.Providers[name][key] = value
		}
	}
	return skipped, invalid
}

// typeMatches reports whether value's concrete Go type (as yaml.v3
// decodes it into a generic map[string]any) matches a provider config
// key's declared type.
func typeMatches(typ string, value any) bool {
	switch typ {
	case "int":
		switch value.(type) {
		case int, int64:
			return true
		}
		return false
	case "bool":
		_, ok := value.(bool)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "map":
		_, ok := value.(map[string]any)
		return ok
	case "list":
		_, ok := value.([]any)
		return ok
	default:
		return true
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func findConfigKey(keys []provider.ConfigKey, key string) (provider.ConfigKey, bool) {
	for _, k := range keys {
		if k.Key == key {
			return k, true
		}
	}
	return provider.ConfigKey{}, false
}
