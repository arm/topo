package core

import (
	"strings"
	"testing"
)

func TestPrintHelp(t *testing.T) {
	out := captureOutput(PrintHelp)
	if !strings.Contains(out, "topo help") {
		t.Fatalf("missing header")
	}
	if !strings.Contains(out, "add-service <compose-filepath> <template-id> [service-name]") {
		t.Fatalf("missing add usage")
	}
}
