package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	nervtask "github.com/war-apps/nerv-task-by-gentle-ai"
	"github.com/war-apps/nerv-task-by-gentle-ai/internal/version"
)

func TestRun_VersionPrintsBinaryAndPlugin(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr = %q", code, stderr.String())
	}
	pluginVersion, err := version.PluginVersion(nervtask.CoreFS())
	if err != nil {
		t.Fatalf("reading plugin version: %v", err)
	}
	want := fmt.Sprintf("nerv-task %s (plugin %s)\n", version.Binary, pluginVersion)
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRun_VersionJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"version", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stderr = %q", code, stderr.String())
	}

	var got struct {
		Binary string `json:"binary"`
		Plugin string `json:"plugin"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v (stdout=%q)", err, stdout.String())
	}
	if got.Binary != version.Binary {
		t.Errorf("json.binary = %q, want %q", got.Binary, version.Binary)
	}
	pluginVersion, err := version.PluginVersion(nervtask.CoreFS())
	if err != nil {
		t.Fatalf("reading plugin version: %v", err)
	}
	if got.Plugin != pluginVersion {
		t.Errorf("json.plugin = %q, want %q", got.Plugin, pluginVersion)
	}
}

func TestRunVersion_UnknownArg_Refuses(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runVersion([]string{"--bogus-flag"}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("runVersion() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("stderr = %q, want it to contain Usage", stderr.String())
	}
}
