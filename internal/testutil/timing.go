package testutil

import (
	"os"
	"testing"
	"time"
)

// MeasureTestPhase enables temporary CI diagnostics without timing assertions.
func MeasureTestPhase(t testing.TB, phase string) func() {
	t.Helper()
	if os.Getenv("TOPO_TEST_TIMINGS") != "1" {
		return func() {}
	}
	start := time.Now()
	return func() {
		t.Helper()
		t.Logf("[DEBUG-perf] %s: %s", phase, time.Since(start))
	}
}
