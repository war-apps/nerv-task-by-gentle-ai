package provider_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

func acmeDirFS(name string) fstest.MapFS {
	return fstest.MapFS{
		name + "/provider.yaml":            &fstest.MapFile{Data: []byte(strings.ReplaceAll(validManifest, "name: acme", "name: acme"))},
		name + "/procedures/get-task.md":   &fstest.MapFile{Data: []byte("x")},
		name + "/procedures/assign.md":     &fstest.MapFile{Data: []byte("x")},
		name + "/procedures/move-stage.md": &fstest.MapFile{Data: []byte("x")},
	}
}

func TestDiscoverEmbedded_FindsTaskProviderFolders(t *testing.T) {
	fsys := fstest.MapFS{}
	for k, v := range acmeDirFS("task-acme") {
		fsys[k] = v
	}
	fsys["not-a-provider/readme.md"] = &fstest.MapFile{Data: []byte("x")}

	manifests, err := provider.DiscoverEmbedded(fsys)
	if err != nil {
		t.Fatalf("DiscoverEmbedded() error = %v", err)
	}
	if len(manifests) != 1 || manifests[0].Name != "acme" {
		t.Fatalf("DiscoverEmbedded() = %v, want exactly one manifest named acme", manifests)
	}
}

func TestDiscoverEmbedded_NoProviders(t *testing.T) {
	manifests, err := provider.DiscoverEmbedded(fstest.MapFS{})
	if err != nil {
		t.Fatalf("DiscoverEmbedded() error = %v", err)
	}
	if len(manifests) != 0 {
		t.Fatalf("DiscoverEmbedded() = %v, want none", manifests)
	}
}

func TestDiscoverProject_FindsProviderUnderAIWorkflow(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".ai-workflow", "providers", "task-acme")
	if err := os.MkdirAll(filepath.Join(dir, "procedures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "provider.yaml"), []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"get-task.md", "assign.md", "move-stage.md"} {
		if err := os.WriteFile(filepath.Join(dir, "procedures", f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	manifests, err := provider.DiscoverProject(root)
	if err != nil {
		t.Fatalf("DiscoverProject() error = %v", err)
	}
	if len(manifests) != 1 || manifests[0].Name != "acme" {
		t.Fatalf("DiscoverProject() = %v, want exactly one manifest named acme", manifests)
	}
}

func TestDiscoverProject_NoProvidersDir(t *testing.T) {
	manifests, err := provider.DiscoverProject(t.TempDir())
	if err != nil {
		t.Fatalf("DiscoverProject() error = %v", err)
	}
	if len(manifests) != 0 {
		t.Fatalf("DiscoverProject() = %v, want none", manifests)
	}
}

func TestDiscover_RefusesNameCollisionBetweenEmbeddedAndProject(t *testing.T) {
	embedded := fstest.MapFS{}
	for k, v := range acmeDirFS("task-acme") {
		embedded[k] = v
	}

	root := t.TempDir()
	dir := filepath.Join(root, ".ai-workflow", "providers", "task-acme")
	if err := os.MkdirAll(filepath.Join(dir, "procedures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "provider.yaml"), []byte(validManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"get-task.md", "assign.md", "move-stage.md"} {
		if err := os.WriteFile(filepath.Join(dir, "procedures", f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := provider.Discover(embedded, root)
	if err == nil {
		t.Fatal("Discover() error = nil, want a name-collision refusal")
	}
	if !strings.Contains(err.Error(), "acme") {
		t.Errorf("Discover() error = %v, want it to name acme", err)
	}
}

func TestDiscover_MergesBoth(t *testing.T) {
	embedded := fstest.MapFS{}
	for k, v := range acmeDirFS("task-acme") {
		embedded[k] = v
	}

	root := t.TempDir()
	dir := filepath.Join(root, ".ai-workflow", "providers", "task-other")
	if err := os.MkdirAll(filepath.Join(dir, "procedures"), 0o755); err != nil {
		t.Fatal(err)
	}
	other := strings.Replace(validManifest, "name: acme", "name: other", 1)
	if err := os.WriteFile(filepath.Join(dir, "provider.yaml"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"get-task.md", "assign.md", "move-stage.md"} {
		if err := os.WriteFile(filepath.Join(dir, "procedures", f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	manifests, err := provider.Discover(embedded, root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	names := make([]string, 0, len(manifests))
	for _, m := range manifests {
		names = append(names, m.Name)
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "acme" || names[1] != "other" {
		t.Fatalf("Discover() names = %v, want [acme other]", names)
	}
}
