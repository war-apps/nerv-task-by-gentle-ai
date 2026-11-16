package main

import (
	"io"
	"os"

	"golang.org/x/term"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"
)

// stdinIsTerminal reports whether os.Stdin is attached to a real
// interactive terminal, via golang.org/x/term's real termios/console
// check (mirrors nerv's cmd/nerv/terminal.go: the std-library-only
// os.ModeCharDevice heuristic misreads a redirected /dev/null as a
// terminal on Windows git-bash).
func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// cliOptions bundles the terminal/input seam every interactive command
// goes through, so tests inject a fixed outcome and a scripted reader
// instead of depending on the real process's actual stdin, plus the
// runner every subprocess call (claude plugin ...) goes through, so
// tests inject a fake instead of shelling out to a real claude binary.
type cliOptions struct {
	IsTerminal func() bool
	Stdin      io.Reader
	Runner     runner.Runner
}

func defaultCLIOptions() cliOptions {
	return cliOptions{
		IsTerminal: stdinIsTerminal,
		Stdin:      os.Stdin,
		Runner:     runner.ExecRunner{},
	}
}

// readAnswerLine reads one line from r a byte at a time, so nothing
// past the line is consumed -- the caller may read the same r again
// right after (mirrors nerv's own readAnswer).
func readAnswerLine(r io.Reader) (string, bool) {
	var line []byte
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n == 1 {
			if buf[0] == '\n' {
				return string(line), true
			}
			line = append(line, buf[0])
		}
		if err != nil {
			return string(line), false
		}
	}
}
