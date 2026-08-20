package podman_test

import (
	"testing"

	gtestutil "github.com/arm/topo/internal/testutil"
)

func sanitiseTestName(t *testing.T) string {
	t.Helper()
	return gtestutil.SanitiseTestName(t)
}

func startContainer(t *testing.T) *gtestutil.Container {
	t.Helper()
	return gtestutil.StartContainer(t, gtestutil.PodmanContainer)
}

func requireWriteFile(t *testing.T, path, content string) {
	t.Helper()
	gtestutil.RequireWriteFile(t, path, content)
}

func requireAvailableTCPPort(t *testing.T) string {
	t.Helper()
	return gtestutil.RequireAvailableTCPPort(t, "127.0.0.1")
}
