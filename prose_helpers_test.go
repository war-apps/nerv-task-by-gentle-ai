package nervtask

import (
	"strings"
	"testing"
)

// assertContainsAll fails the test, naming every substring missing from
// text, with path identifying the source for the failure message.
func assertContainsAll(t *testing.T, path, text string, substrings ...string) {
	t.Helper()
	for _, s := range substrings {
		if !strings.Contains(text, s) {
			t.Errorf("%s does not contain %q", path, s)
		}
	}
}
