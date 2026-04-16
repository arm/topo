package e2e

import "testing"

func Step(t testing.TB, desc string) {
	t.Helper()
	t.Log("step:", desc)
}
