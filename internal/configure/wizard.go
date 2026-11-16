package configure

import (
	"fmt"
	"io"
	"strconv"

	"github.com/war-apps/nerv-task-by-gentle-ai/internal/config"
)

// WizardResult reports what RunWizard did.
type WizardResult struct {
	Changed   bool
	Backup    string
	ChangeLog []string
}

// RunWizard drives nerv-task configure's interactive wizard: for every
// known provider (deps.Providers: the embedded ones, plus any project
// provider), asks whether to enable it now, and for each enabled
// provider's scope: user keys, prompts with the manifest's prompt and
// current value or default (Enter keeps it). It then asks timer_store
// and stale_after_hours the same way, shows a diff of every changed
// key, and writes the global task.yaml only on confirmation.
//
// in is read one line at a time: a real terminal, or an --answers file
// (one answer per line). A genuinely exhausted reader before the wizard
// finishes returns ErrInputClosed and writes nothing -- a blank line is
// always "keep current", never treated the same as EOF.
func RunWizard(deps Deps, in io.Reader, out io.Writer) (WizardResult, error) {
	s := newSession(in, out)

	cfg, _, err := config.LoadGlobal(deps.GlobalPath)
	if err != nil {
		return WizardResult{}, err
	}

	fmt.Fprintln(out, "--- nerv-task configuration ---")
	fmt.Fprintf(out, "File: %s\n\n", deps.GlobalPath)

	var changeLog []string
	record := func(label, oldValue, newValue string) {
		if newValue != oldValue {
			changeLog = append(changeLog, fmt.Sprintf("%s: %s -> %s", label, dashIfEmpty(oldValue), dashIfEmpty(newValue)))
		}
	}

	timerStoreDefault := cfg.TimerStore
	if timerStoreDefault == "" {
		timerStoreDefault = config.DefaultTimerStore
	}
	newTimerStore, err := s.ask("Timer store path", timerStoreDefault)
	if err != nil {
		return WizardResult{}, err
	}
	record("timer_store", timerStoreDefault, newTimerStore)
	cfg.TimerStore = newTimerStore

	staleCurrent := strconv.Itoa(config.DefaultStaleAfterHours)
	if cfg.StaleAfterHours != nil {
		staleCurrent = strconv.Itoa(*cfg.StaleAfterHours)
	}
	staleAnswer, err := askTyped(s, "Stale timer threshold (hours)", "int", staleCurrent)
	if err != nil {
		return WizardResult{}, err
	}
	if staleAnswer != staleCurrent {
		n, convErr := strconv.Atoi(staleAnswer)
		if convErr == nil {
			cfg.StaleAfterHours = &n
		}
	}
	record("stale_after_hours", staleCurrent, staleAnswer)

	fmt.Fprintln(out)
	fmt.Fprintln(out, "--- Providers ---")
	if len(deps.Providers) == 0 {
		fmt.Fprintln(out, "No task provider is embedded; nothing to configure.")
	}
	for _, m := range deps.Providers {
		defaultEnable := len(deps.Providers) == 1 || len(cfg.Providers[m.Name]) > 0
		enabled, err := s.yesNo(fmt.Sprintf("Enable %s?", m.Name), defaultEnable)
		if err != nil {
			return WizardResult{}, err
		}
		if !enabled {
			continue
		}
		fmt.Fprintln(out)
		fmt.Fprintf(out, "-- %s --\n", m.Name)
		for _, c := range m.Config {
			if c.Scope != "user" || !c.Scalar() {
				continue
			}
			old := ""
			if v := cfg.ProviderValue(m.Name, c.Key); v != nil {
				old = fmt.Sprint(v)
			}
			def := old
			if def == "" {
				def = c.Default
			}
			answer, err := askTyped(s, c.Prompt, c.Type, def)
			if err != nil {
				return WizardResult{}, err
			}
			if answer == def {
				continue
			}
			typedValue, convErr := typedConfigValue(c.Type, answer)
			if convErr != nil {
				continue // already validated by askTyped; defensive only
			}
			if cfg.Providers == nil {
				cfg.Providers = map[string]map[string]any{}
			}
			if cfg.Providers[m.Name] == nil {
				cfg.Providers[m.Name] = map[string]any{}
			}
			cfg.Providers[m.Name][c.Key] = typedValue
			record(m.Name+"."+c.Key, old, fmt.Sprint(typedValue))
		}
	}

	fmt.Fprintln(out)
	if len(changeLog) == 0 {
		fmt.Fprintf(out, "No changes; %s left untouched.\n", deps.GlobalPath)
		return WizardResult{}, nil
	}
	fmt.Fprintln(out, "--- Summary ---")
	for _, c := range changeLog {
		fmt.Fprintf(out, "  %s\n", c)
	}
	write, err := s.yesNo(fmt.Sprintf("Write to %s?", deps.GlobalPath), true)
	if err != nil {
		return WizardResult{}, err
	}
	if !write {
		fmt.Fprintln(out, "Aborted; nothing written.")
		return WizardResult{ChangeLog: changeLog}, nil
	}

	written, backup, err := config.SaveGlobal(deps.GlobalPath, cfg, deps.now())
	if err != nil {
		return WizardResult{}, err
	}
	if backup != "" {
		fmt.Fprintf(out, "Backup written: %s\n", backup)
	}
	if written {
		fmt.Fprintf(out, "Written: %s\n", deps.GlobalPath)
	}
	return WizardResult{Changed: written, Backup: backup, ChangeLog: changeLog}, nil
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
