package nervtask

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/provider"
)

// Every provider nerv-task ships must satisfy the contract: a valid
// manifest whose operation files exist inside its own plugin folder.
// Ported from nerv's internal/taskprovider/embedded_test.go
// (TestEmbeddedProviders_AreValid).
func TestEmbeddedProviders_AreValid(t *testing.T) {
	manifests, err := provider.DiscoverEmbedded(ExtensionsFS())
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) == 0 {
		t.Fatal("no embedded provider: the Teamwork provider must ship")
	}
	for _, m := range manifests {
		root, err := fs.Sub(ExtensionsFS(), "task-"+m.Name)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.CheckFiles(root); err != nil {
			t.Errorf("task-%s: %v", m.Name, err)
		}
	}
}

// The Teamwork provider calls Teamwork's official MCP server: it ships
// exactly that server, with no credential, in its own .mcp.json.
func TestTeamworkProvider_ShipsOnlyTheOfficialServerWithNoCredential(t *testing.T) {
	m, err := provider.Load(ExtensionsFS(), "task-teamwork")
	if err != nil {
		t.Fatal(err)
	}
	if m.MCP.Server != "teamwork" {
		t.Errorf("mcp.server = %q, want teamwork", m.MCP.Server)
	}

	data, err := fs.ReadFile(ExtensionsFS(), "task-teamwork/.mcp.json")
	if err != nil {
		t.Fatalf("task-teamwork ships no .mcp.json: %v", err)
	}
	text := string(data)
	assertContainsAll(t, "task-teamwork/.mcp.json", text, `"teamwork"`, `"http"`, "https://mcp.ai.teamwork.com")
	if strings.Contains(text, "Authorization") {
		t.Error(".mcp.json carries a credential header; the server is reached with OAuth, no credential in the plugin")
	}
}
