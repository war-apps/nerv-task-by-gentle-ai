package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeTerminalOptions(terminal bool, answers string) cliOptions {
	return cliOptions{
		IsTerminal: func() bool { return terminal },
		Stdin:      strings.NewReader(answers),
	}
}

func TestRunConfigure_Terminal_NoFlags_RunsWizard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	// timer_store, stale_after_hours, "Enable teamwork?" and every one
	// of its scope: user keys -> all blank (keep defaults); nothing
	// changes, nothing written. Generously many blank lines so this
	// does not depend on exactly how many config keys teamwork declares.
	opts := fakeTerminalOptions(true, strings.Repeat("\n", 20))
	var stdout, stderr bytes.Buffer
	code := runConfigureWithOptions(nil, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runConfigureWithOptions() code = %d, stderr = %q, stdout = %q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "nerv-task configuration") {
		t.Errorf("stdout = %q, want the wizard banner", stdout.String())
	}
}

func TestRunConfigure_NonTerminal_NoFlags_PrintsCurrentConfig_NoPrompts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	opts := fakeTerminalOptions(false, "")
	var stdout, stderr bytes.Buffer
	code := runConfigureWithOptions(nil, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runConfigureWithOptions() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "config_path:") {
		t.Errorf("stdout = %q, want the printed configuration", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--set") {
		t.Errorf("stdout = %q, want a hint on how to set a key", stdout.String())
	}
}

func TestRunConfigure_Answers_DrivesWizardWithoutATerminal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	answersFile := filepath.Join(t.TempDir(), "answers.txt")
	// All blank: keep every default, change nothing. Generously many
	// lines so this does not depend on exactly how many config keys
	// teamwork declares.
	content := strings.Repeat("\n", 20)
	if err := os.WriteFile(answersFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := fakeTerminalOptions(false, "") // not a terminal; --answers must still drive it
	var stdout, stderr bytes.Buffer
	code := runConfigureWithOptions([]string{"--answers", answersFile}, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runConfigureWithOptions() code = %d, stderr = %q, stdout = %q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "nerv-task configuration") {
		t.Errorf("stdout = %q, want the wizard banner", stdout.String())
	}
}

func TestRunConfigure_Answers_MissingFile_TreatedAsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	opts := fakeTerminalOptions(false, "")
	var stdout, stderr bytes.Buffer
	code := runConfigureWithOptions([]string{"--answers", filepath.Join(t.TempDir(), "missing.txt")}, &stdout, &stderr, opts)
	// An empty answers file hits EOF on the very first prompt: ErrInputClosed.
	if code != 1 {
		t.Fatalf("runConfigureWithOptions() code = %d, want 1 (input closed); stderr = %q", code, stderr.String())
	}
}

func TestRunConfigure_PrintFlag_IgnoresTerminalAndAnswers(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	opts := fakeTerminalOptions(true, "")
	var stdout, stderr bytes.Buffer
	code := runConfigureWithOptions([]string{"--print"}, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runConfigureWithOptions() code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "nerv-task configuration") {
		t.Error("--print must never run the interactive wizard")
	}
}
