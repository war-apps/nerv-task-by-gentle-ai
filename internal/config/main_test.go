package config_test

import (
	"os"
	"testing"
)

// TestMain clears XDG_CONFIG_HOME so the user config path is built from
// each test's fake home, not from the machine running the tests (GitHub's
// Linux runners set it).
func TestMain(m *testing.M) {
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Exit(m.Run())
}
