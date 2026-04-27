package e2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireWriteDummyExecutable(t testing.TB, path string) {
	t.Helper()
	err := os.WriteFile(path, nil, 0o755)
	require.NoError(t, err)
}
