package run

import (
	"testing"
)

func TestParseCliDefaultsToHelp(t *testing.T) {
	cli := ParseCli()
	// when no args (os.Args from test runner has the test binary name only), ParseCli should set help
	if cli.Command != "help" && cli.Command != "" {
		t.Fatalf("expected help or empty, got %q", cli.Command)
	}
}
