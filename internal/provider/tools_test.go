package provider_test

import (
	"reflect"
	"testing"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

func acmeManifest(t *testing.T) *provider.Manifest {
	t.Helper()
	m, err := provider.Parse([]byte(validManifest))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMissingTools_BareNameCounts(t *testing.T) {
	m := acmeManifest(t)
	got := m.MissingTools([]string{"acme-get_task", "acme-assign", "acme-move_stage"})
	if len(got) != 0 {
		t.Errorf("MissingTools() = %v, want none missing", got)
	}
}

func TestMissingTools_ServerPrefixedNameCounts(t *testing.T) {
	m := acmeManifest(t)
	got := m.MissingTools([]string{
		"mcp__acme__acme-get_task",
		"mcp__acme__acme-assign",
		"mcp__acme__acme-move_stage",
	})
	if len(got) != 0 {
		t.Errorf("MissingTools() = %v, want none missing", got)
	}
}

func TestMissingTools_PluginPrefixedNameCounts_DashAndUnderscoreVariants(t *testing.T) {
	m := acmeManifest(t)
	got := m.MissingTools([]string{
		"mcp__plugin_task-acme_acme__acme-get_task",
		"mcp__plugin_task_acme_acme__acme-assign",
		"mcp__plugin_task-acme_acme__acme-move_stage",
	})
	if len(got) != 0 {
		t.Errorf("MissingTools() = %v, want none missing", got)
	}
}

func TestMissingTools_ConnectorPrefixedNameCounts(t *testing.T) {
	m := acmeManifest(t)
	got := m.MissingTools([]string{
		"mcp__claude_ai_Acme_com__acme-get_task",
		"mcp__claude_ai_Acme_com__acme-assign",
		"mcp__claude_ai_Acme_com__acme-move_stage",
	})
	if len(got) != 0 {
		t.Errorf("MissingTools() = %v, want none missing", got)
	}
}

func TestMissingTools_UndeclaredConnectorDoesNotCount(t *testing.T) {
	m := acmeManifest(t)
	got := m.MissingTools([]string{"mcp__claude_ai_Other_com__acme-get_task"})
	want := []string{"acme-get_task", "acme-assign", "acme-move_stage"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MissingTools() = %v, want %v", got, want)
	}
}

func TestMissingTools_SameToolFromAnotherServerDoesNotCount(t *testing.T) {
	m := acmeManifest(t)
	got := m.MissingTools([]string{"mcp__other-server__acme-get_task"})
	want := []string{"acme-get_task", "acme-assign", "acme-move_stage"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MissingTools() = %v, want %v", got, want)
	}
}

func TestMissingTools_ReportsEachMissingInDeclarationOrder(t *testing.T) {
	m := acmeManifest(t)
	got := m.MissingTools([]string{"acme-get_task"})
	want := []string{"acme-assign", "acme-move_stage"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MissingTools() = %v, want %v", got, want)
	}
}

func TestConnectorToolPrefix(t *testing.T) {
	cases := map[string]string{
		"Acme.com":     "mcp__claude_ai_Acme_com__",
		"Teamwork.com": "mcp__claude_ai_Teamwork_com__",
	}
	for name, want := range cases {
		if got := provider.ConnectorToolPrefix(name); got != want {
			t.Errorf("ConnectorToolPrefix(%q) = %q, want %q", name, got, want)
		}
	}
}
