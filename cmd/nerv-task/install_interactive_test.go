package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner/runnertest"
)

func fakeInstallOptions(terminal bool, answers string) cliOptions {
	return cliOptions{
		IsTerminal: func() bool { return terminal },
		Stdin:      strings.NewReader(answers),
		Runner:     &runnertest.FakeRunner{},
	}
}

func TestRunInstall_Terminal_OffersConfigure_YesRunsWizard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	// "Run nerv-task configure now?" -> y, then enough blanks for the
	// wizard to run to completion without changing anything.
	opts := fakeInstallOptions(true, "y\n"+strings.Repeat("\n", 20))
	var stdout, stderr bytes.Buffer
	code := runInstallWithOptions([]string{"--with-teamwork"}, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runInstallWithOptions() code = %d, stderr = %q, stdout = %q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "nerv-task configuration") {
		t.Errorf("stdout = %q, want the wizard banner after accepting the offer", stdout.String())
	}
}

func TestRunInstall_Terminal_OffersConfigure_NoSkipsWizard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	opts := fakeInstallOptions(true, "n\n")
	var stdout, stderr bytes.Buffer
	code := runInstallWithOptions(nil, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runInstallWithOptions() code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "nerv-task configuration") {
		t.Error("declining the offer must never run the wizard")
	}
}

func TestRunInstall_NoConfigureFlag_NeverOffers(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	opts := fakeInstallOptions(true, "")
	var stdout, stderr bytes.Buffer
	code := runInstallWithOptions([]string{"--no-configure"}, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runInstallWithOptions() code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "Run nerv-task configure now?") {
		t.Error("--no-configure must never offer to run configure")
	}
	if strings.Contains(stdout.String(), "nerv-task configuration") {
		t.Error("--no-configure must never run the wizard")
	}
}

func TestRunInstall_NonTerminal_PrintsHintInsteadOfOffering(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	opts := fakeInstallOptions(false, "")
	var stdout, stderr bytes.Buffer
	code := runInstallWithOptions(nil, &stdout, &stderr, opts)
	if code != 0 {
		t.Fatalf("runInstallWithOptions() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nerv-task configure") {
		t.Errorf("stdout = %q, want a hint to run nerv-task configure later", stdout.String())
	}
	if strings.Contains(stdout.String(), "nerv-task configuration") {
		t.Error("a non-terminal run must never run the wizard")
	}
}
