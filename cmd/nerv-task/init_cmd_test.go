package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_None_WritesNothing(t *testing.T) {
	repo := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := runInit([]string{"--none"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runInit() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nothing written") {
		t.Errorf("stdout = %q, want it to mention nothing written", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".ai-workflow", "task.yaml")); err == nil {
		t.Error("task.yaml was written, want nothing for --none")
	}
}

func TestRunInit_ExplicitProvider_WritesTaskYaml(t *testing.T) {
	repo := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := runInit([]string{"--provider", "teamwork"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runInit() code = %d, stderr = %q", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(repo, ".ai-workflow", "task.yaml"))
	if err != nil {
		t.Fatalf("task.yaml was not written: %v", err)
	}
	if !strings.Contains(string(data), "teamwork") {
		t.Errorf("task.yaml = %q, want it to mention teamwork", string(data))
	}
}
