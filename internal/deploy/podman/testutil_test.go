package podman_test

import (
	"testing"

	deploytestutil "github.com/arm/topo/internal/deploy/testutil"
	gtestutil "github.com/arm/topo/internal/testutil"
)

func unmarshalNDJSON(ndJSON []byte) ([]deploytestutil.JsonObject, error) {
	return deploytestutil.UnmarshalNDJSON(ndJSON)
}

func sanitiseTestName(t *testing.T) string {
	t.Helper()
	return gtestutil.SanitiseTestName(t)
}

func requireWriteFile(t *testing.T, path, content string) {
	t.Helper()
	gtestutil.RequireWriteFile(t, path, content)
}
