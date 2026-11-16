package provider

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// providerDirPrefix is how every task provider's own folder is named,
// both embedded (plugin/extensions/task-<name>) and project-local
// (.ai-workflow/providers/task-<name>).
const providerDirPrefix = "task-"

// ProjectProvidersDir is where a repository ships its own task
// providers, committed with the repository and never installed.
const ProjectProvidersDir = ".ai-workflow/providers"

// DiscoverEmbedded finds every task-<name> folder at the top of fsys that
// holds a provider.yaml, and loads it.
func DiscoverEmbedded(fsys fs.FS) ([]*Manifest, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded providers: %w", err)
	}
	var manifests []*Manifest
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), providerDirPrefix) {
			continue
		}
		if _, err := fs.Stat(fsys, path.Join(e.Name(), ManifestFile)); err != nil {
			continue
		}
		m, err := Load(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, m)
	}
	sort.Slice(manifests, func(i, j int) bool { return manifests[i].Name < manifests[j].Name })
	return manifests, nil
}

// DiscoverProject finds every task-<name> folder under
// <root>/.ai-workflow/providers that holds a provider.yaml, and loads it.
// A missing providers directory is not an error: it simply yields none.
func DiscoverProject(root string) ([]*Manifest, error) {
	dir := filepath.Join(root, ProjectProvidersDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var manifests []*Manifest
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), providerDirPrefix) {
			continue
		}
		manifestPath := filepath.Join(dir, e.Name(), ManifestFile)
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}
		m, err := Load(os.DirFS(filepath.Join(dir, e.Name())), ".")
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, m)
	}
	sort.Slice(manifests, func(i, j int) bool { return manifests[i].Name < manifests[j].Name })
	return manifests, nil
}

// Discover merges the embedded providers with the repository's own
// project providers (root may be ""), refusing a name collision between
// them.
func Discover(embedded fs.FS, root string) ([]*Manifest, error) {
	fromEmbedded, err := DiscoverEmbedded(embedded)
	if err != nil {
		return nil, err
	}
	var fromProject []*Manifest
	if root != "" {
		fromProject, err = DiscoverProject(root)
		if err != nil {
			return nil, err
		}
	}

	byName := map[string]bool{}
	for _, m := range fromEmbedded {
		byName[m.Name] = true
	}
	var merged []*Manifest
	merged = append(merged, fromEmbedded...)
	for _, m := range fromProject {
		if byName[m.Name] {
			return nil, fmt.Errorf("provider %s: a project provider collides with an embedded one of the same name", m.Name)
		}
		merged = append(merged, m)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Name < merged[j].Name })
	return merged, nil
}
