package provider

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var taskRefPrefixRe = regexp.MustCompile(`^[A-Z][A-Z0-9]*$`)

// MCP names the server and the exact tools the provider calls, so a
// dispatching agent can tell a missing connector from a provider failure.
type MCP struct {
	Server     string   `yaml:"server"`
	Tools      []string `yaml:"tools"`
	Connectors []string `yaml:"connectors"`
}

// ConfigKey is one setting the provider needs, with what nerv-task
// configure shows for it. Scope decides which file holds the value:
// "repo" lives in the repository's task.yaml, "user" in the global
// task.yaml.
type ConfigKey struct {
	Key     string `yaml:"key"`
	Scope   string `yaml:"scope"`
	Type    string `yaml:"type"`
	Default string `yaml:"default"`
	Prompt  string `yaml:"prompt"`
}

// Scalar reports whether the key's type is a plain scalar (string, int or
// bool), as opposed to a map or a list.
func (c ConfigKey) Scalar() bool {
	return c.Type == "string" || c.Type == "int" || c.Type == "bool"
}

// Manifest is a task provider's provider.yaml, as nerv-task reads it.
type Manifest struct {
	Name            string
	ContractVersion int
	Runtime         string
	TaskRefPrefix   string
	MCP             MCP
	Config          []ConfigKey
	Stages          map[string]string
	// Operations maps every operation to its procedure file, or to
	// NotImplemented.
	Operations map[string]string
	Events     []string
}

// rawManifest is the YAML shape of provider.yaml, decoded before
// validation.
type rawManifest struct {
	Name            string            `yaml:"name"`
	ContractVersion int               `yaml:"contract_version"`
	Runtime         string            `yaml:"runtime"`
	TaskRefPrefix   string            `yaml:"task_ref_prefix"`
	MCP             MCP               `yaml:"mcp"`
	Config          []ConfigKey       `yaml:"config"`
	Stages          map[string]string `yaml:"stages"`
	Operations      map[string]string `yaml:"operations"`
	Events          []string          `yaml:"events"`
}

// Parse decodes and validates a provider.yaml document. Every problem
// found is reported at once, joined with "; ".
func Parse(data []byte) (*Manifest, error) {
	var raw rawManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", ManifestFile, err)
	}

	m := &Manifest{
		Name:            raw.Name,
		ContractVersion: raw.ContractVersion,
		Runtime:         raw.Runtime,
		TaskRefPrefix:   raw.TaskRefPrefix,
		MCP:             raw.MCP,
		Config:          raw.Config,
		Stages:          raw.Stages,
		Operations:      raw.Operations,
		Events:          raw.Events,
	}

	problems := m.validate()
	if len(problems) > 0 {
		return nil, errors.New(ManifestFile + ": " + strings.Join(problems, "; "))
	}
	return m, nil
}

// Load reads and validates dir/provider.yaml from fsys.
func Load(fsys fs.FS, dir string) (*Manifest, error) {
	data, err := fs.ReadFile(fsys, path.Join(dir, ManifestFile))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path.Join(dir, ManifestFile), err)
	}
	return Parse(data)
}

// Implements reports whether the provider implements op.
func (m *Manifest) Implements(op string) bool {
	file, ok := m.Operations[op]
	return ok && file != NotImplemented
}

// Subscribes reports whether the provider handles event.
func (m *Manifest) Subscribes(event string) bool {
	return slices.Contains(m.Events, event)
}

// CheckFiles reports every operation file the manifest names that fsys
// (the provider's own folder) does not hold.
func (m *Manifest) CheckFiles(fsys fs.FS) error {
	var missing []string
	for _, op := range Operations {
		if !m.Implements(op) {
			continue
		}
		file := m.Operations[op]
		if _, err := fs.Stat(fsys, file); err != nil {
			missing = append(missing, fmt.Sprintf("operations.%s: %s not found", op, file))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("provider %s: %s", m.Name, strings.Join(missing, "; "))
	}
	return nil
}

func (m *Manifest) validate() []string {
	var p []string
	add := func(format string, args ...any) { p = append(p, fmt.Sprintf(format, args...)) }

	if m.Name == "" {
		add("name is required")
	}
	if m.ContractVersion != ContractVersion {
		add("contract_version %d, want %d", m.ContractVersion, ContractVersion)
	}
	if m.Runtime == "" {
		m.Runtime = RuntimeAgent
	}
	if m.Runtime != RuntimeAgent {
		add("runtime %q: nerv-task providers run as agent, nothing else", m.Runtime)
	}
	if !taskRefPrefixRe.MatchString(m.TaskRefPrefix) {
		add("task_ref_prefix %q must be upper-case letters and digits", m.TaskRefPrefix)
	}

	p = append(p, m.validateMCP()...)
	p = append(p, m.validateConfig()...)
	p = append(p, m.validateStages()...)
	p = append(p, m.validateOperations()...)
	p = append(p, m.validateEvents()...)

	_ = add
	return p
}

func (m *Manifest) validateMCP() []string {
	var p []string
	if m.MCP.Server == "" {
		p = append(p, "mcp.server is required")
	}
	if len(m.MCP.Tools) == 0 {
		p = append(p, "mcp.tools must list at least one tool")
	}
	seen := map[string]bool{}
	for _, tool := range m.MCP.Tools {
		if seen[tool] {
			p = append(p, fmt.Sprintf("mcp.tools: %s is declared more than once", tool))
		}
		seen[tool] = true
	}
	seenConn := map[string]bool{}
	for _, c := range m.MCP.Connectors {
		if seenConn[c] {
			p = append(p, fmt.Sprintf("mcp.connectors: %s is declared more than once", c))
		}
		seenConn[c] = true
	}
	return p
}

func (m *Manifest) validateConfig() []string {
	var p []string
	seen := map[string]bool{}
	for _, c := range m.Config {
		if c.Key == "" {
			p = append(p, "config: a key is required")
			continue
		}
		if seen[c.Key] {
			p = append(p, fmt.Sprintf("config: %s is declared more than once", c.Key))
		}
		seen[c.Key] = true
		if !slices.Contains(ConfigScopes, c.Scope) {
			p = append(p, fmt.Sprintf("config.%s: scope %q must be one of %s", c.Key, c.Scope, strings.Join(ConfigScopes, ", ")))
		}
		if !slices.Contains(ConfigTypes, c.Type) {
			p = append(p, fmt.Sprintf("config.%s: type %q must be one of %s", c.Key, c.Type, strings.Join(ConfigTypes, ", ")))
			continue
		}
		if c.Default != "" && c.Scalar() && !validValue(c.Type, c.Default) {
			p = append(p, fmt.Sprintf("config.%s: default %q is not a valid %s", c.Key, c.Default, c.Type))
		}
	}
	return p
}

func validValue(typ, value string) bool {
	switch typ {
	case "int":
		_, err := strconv.Atoi(value)
		return err == nil
	case "bool":
		_, err := strconv.ParseBool(value)
		return err == nil
	}
	return true
}

func (m *Manifest) validateStages() []string {
	var p []string
	keys := make([]string, 0, len(m.Stages))
	for k := range m.Stages {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if !slices.Contains(StageKeys, key) {
			p = append(p, fmt.Sprintf("stages.%s is not a canonical stage (%s)", key, strings.Join(StageKeys, ", ")))
		}
	}
	return p
}

func (m *Manifest) validateOperations() []string {
	var p []string
	for _, op := range Operations {
		file, ok := m.Operations[op]
		if !ok {
			p = append(p, fmt.Sprintf("operations.%s is required (a path, or %s)", op, NotImplemented))
			continue
		}
		if file == NotImplemented {
			continue
		}
		if !strings.HasSuffix(file, ".md") || path.IsAbs(file) || strings.Contains(file, "..") {
			p = append(p, fmt.Sprintf("operations.%s: %q must be a relative .md path with no ..", op, file))
		}
	}
	return p
}

func (m *Manifest) validateEvents() []string {
	var p []string
	seen := map[string]bool{}
	for _, event := range m.Events {
		if seen[event] {
			p = append(p, fmt.Sprintf("events: %s is declared more than once", event))
		}
		seen[event] = true
		needs, ok := Events[event]
		if !ok {
			p = append(p, fmt.Sprintf("events: %s is not a known pipeline event", event))
			continue
		}
		for _, op := range needs {
			if !m.Implements(op) {
				p = append(p, fmt.Sprintf("events: %s needs %s, declared %s", event, op, NotImplemented))
			}
		}
	}
	return p
}
