package project_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestCheckEnvCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name       string
		parameters string
		args       string
		wantLegacy bool
	}{
		{"literal value", "FOO: {}", `{FOO: bar}`, true},
		{"typed value", "FOO: {}", `{FOO: 123}`, true},
		{"shorthand arguments", "FOO: {}", `[FOO=bar]`, true},
		{"environment reference", "FOO: {}", `{FOO: "${FOO}"}`, false},
		{"required reference", "FOO: {}", `{FOO: "${FOO?configured via topo}"}`, false},
		{"default reference", "FOO: {}", `{FOO: "${FOO:-bar}"}`, false},
		{"unbraced reference", "FOO: {}", `{FOO: "$FOO"}`, false},
		{"escaped reference is literal", "FOO: {}", `{FOO: "$${FOO}"}`, true},
		{"unused parameter", "OTHER: {}", `{FOO: bar}`, false},
		{"no parameters", "", `{FOO: bar}`, false},
		{"partially referenced parameters", "FOO: {}, OTHER: {}", `{FOO: bar, OTHER: "${OTHER}"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// No build context: detection must not require modern Compose validation.
			contents := fmt.Sprintf(`services:
  app:
    build:
      args: %s
x-topo:
  parameters: {%s}
`, tc.args, tc.parameters)
			root := t.TempDir()
			path := testutil.RequireWriteComposeFile(t, root, contents)

			err := project.CheckEnvCompatibility(path)

			if tc.wantLegacy {
				require.ErrorIs(t, err, project.ErrLegacyParameterFormat)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, contents, testutil.RequireReadFile(t, path))
			require.NoFileExists(t, filepath.Join(root, env.DefaultFilename))
		})
	}

	t.Run("propagates inspection errors", func(t *testing.T) {
		err := project.CheckEnvCompatibility(filepath.Join(t.TempDir(), "missing.yaml"))

		require.Error(t, err)
		require.NotErrorIs(t, err, project.ErrLegacyParameterFormat)
	})
}
