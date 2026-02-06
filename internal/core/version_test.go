package core

import (
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	out := captureOutput(PrintVersion)
	expected := strings.TrimSpace(VersionTxt) + "\n"
	if out != expected {
		t.Fatalf("version mismatch: %q vs %q", expected, out)
	}
}
