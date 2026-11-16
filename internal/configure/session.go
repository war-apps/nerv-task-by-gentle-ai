package configure

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

// maxInvalidAttempts bounds every re-prompt loop below at 3 attempts, so
// a scripted or exhausted reader can never spin forever on invalid
// input: once exhausted, the prompt falls back to its "keep current"
// answer instead. Mirrors nerv's own wizard (internal/wizard/prompt.go).
const maxInvalidAttempts = 3

// ErrInputClosed is returned when the underlying reader reaches genuine
// EOF before the wizard finished, as opposed to a blank line, which is
// still "keep current". Mirrors nerv's own wizard: a redirected
// /dev/null or an exhausted --answers file must abort rather than
// silently accept every default and "yes" to every confirmation.
var ErrInputClosed = errors.New("input ended before the wizard finished; nothing was written")

// session drives every prompt in one wizard run from a single shared
// line-based reader, so answers (whether typed live or supplied by an
// --answers file) are consumed in order exactly once.
type session struct {
	in  *bufio.Scanner
	out io.Writer
}

func newSession(in io.Reader, out io.Writer) *session {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 4096), 1<<20)
	return &session{in: scanner, out: out}
}

// readLine returns the next line of input and eof=false, or eof=true
// once the reader is genuinely exhausted (as opposed to a blank line,
// which Scan still reports as a successful read of the empty string).
func (s *session) readLine() (line string, eof bool) {
	if s.in.Scan() {
		return s.in.Text(), false
	}
	return "", true
}

// ask prompts label pre-filled with current; a blank answer keeps
// current. Returns ErrInputClosed if the reader is genuinely exhausted
// rather than silently keeping current.
func (s *session) ask(label, current string) (string, error) {
	fmt.Fprintf(s.out, "%s [%s]: ", label, current)
	line, eof := s.readLine()
	if eof {
		return "", ErrInputClosed
	}
	ans := strings.TrimSpace(line)
	if ans == "" {
		return current, nil
	}
	return ans, nil
}

// choose prompts label with the allowed values and current pre-filled,
// re-prompting on an invalid non-blank answer up to maxInvalidAttempts
// times before giving up and keeping current. Returns ErrInputClosed if
// the reader is genuinely exhausted.
func (s *session) choose(label string, allowed []string, current string) (string, error) {
	for attempt := 0; attempt < maxInvalidAttempts; attempt++ {
		fmt.Fprintf(s.out, "%s (%s) [%s]: ", label, strings.Join(allowed, "|"), current)
		line, eof := s.readLine()
		if eof {
			return "", ErrInputClosed
		}
		ans := strings.TrimSpace(line)
		if ans == "" {
			return current, nil
		}
		if slices.Contains(allowed, ans) {
			return ans, nil
		}
		fmt.Fprintf(s.out, "Invalid input. Allowed: %s.\n", strings.Join(allowed, ", "))
	}
	return current, nil
}

// yesRe matches a "yes" confirmation answer, case-insensitively.
var yesRe = regexp.MustCompile(`(?i)^y(es)?$`)

// yesNo prompts a [Y/n] (defaultYes) or [y/N] confirmation; a blank
// answer resolves to defaultYes. Returns ErrInputClosed if the reader is
// genuinely exhausted rather than silently resolving to defaultYes.
func (s *session) yesNo(label string, defaultYes bool) (bool, error) {
	hint := "y/N"
	if defaultYes {
		hint = "Y/n"
	}
	fmt.Fprintf(s.out, "%s [%s]: ", label, hint)
	line, eof := s.readLine()
	if eof {
		return false, ErrInputClosed
	}
	ans := strings.TrimSpace(line)
	if ans == "" {
		return defaultYes, nil
	}
	return yesRe.MatchString(ans), nil
}

// askTyped prompts label, pre-filled with current, validating the
// answer against typ: a bool is a true|false choice, anything else is
// re-prompted (up to maxInvalidAttempts times, then current is kept)
// until it parses as typ. Mirrors nerv's own askProviderKey.
func askTyped(s *session, label, typ, current string) (string, error) {
	if typ == "bool" {
		return s.choose(label, []string{"true", "false"}, current)
	}
	for attempt := 0; attempt < maxInvalidAttempts; attempt++ {
		answer, err := s.ask(label, current)
		if err != nil {
			return "", err
		}
		if answer == "" {
			return current, nil
		}
		if _, err := typedConfigValue(typ, answer); err == nil {
			return answer, nil
		}
		fmt.Fprintf(s.out, "Invalid input. Expected %s.\n", typ)
	}
	return current, nil
}
