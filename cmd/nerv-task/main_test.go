package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_NoArgs_PrintsUsageAndRefuses(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 2 {
		t.Errorf("run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("stderr = %q, want it to contain Usage", stderr.String())
	}
}

func TestRun_UnknownCommand_Refuses(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"bogus"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "bogus") {
		t.Errorf("stderr = %q, want it to name the unknown command", stderr.String())
	}
}

func TestRun_Help_Succeeds(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("run() code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "nerv-task") {
		t.Errorf("stdout = %q, want it to mention nerv-task", stdout.String())
	}
}

func TestRunInstall_InvalidFlag_Refuses(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runInstall([]string{"--bogus-flag"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("runInstall() code = %d, want 2", code)
	}
}

func TestRunConfigure_InvalidFlag_Refuses(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runConfigure([]string{"--bogus-flag"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("runConfigure() code = %d, want 2", code)
	}
}

func TestRunConfigure_Print_Succeeds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := runConfigure([]string{"--print"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runConfigure() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "config_path:") {
		t.Errorf("stdout = %q, want it to contain config_path:", stdout.String())
	}
	if !strings.Contains(stdout.String(), "teamwork") {
		t.Errorf("stdout = %q, want it to list the embedded teamwork provider", stdout.String())
	}
}

func TestRunConfigure_PrintJSON_Succeeds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	var stdout, stderr bytes.Buffer
	code := runConfigure([]string{"--print", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runConfigure() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"config_path"`) {
		t.Errorf("stdout = %q, want JSON with config_path", stdout.String())
	}
}

// The top-level usage names each command with the same flags as the
// command's own usage line.
func TestUsage_MatchesEachCommandsOwnUsageLine(t *testing.T) {
	for _, own := range []string{configureUsage} {
		first := strings.TrimPrefix(strings.SplitN(own, "\n", 2)[0], "Usage: ")
		if !strings.Contains(usage, "  "+first+"\n") {
			t.Errorf("top-level usage lacks %q:\n%s", first, usage)
		}
	}
}
