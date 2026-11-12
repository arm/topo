package project_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureLiteralExtends(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "accepts the calling project's environment", value: "hello"},
		{name: "rejects a missing value despite dotenv beside the extended file", wantError: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			testutil.Unsetenv(t, "TOPO_TEST_EXTENDS_GREETING")
			root := t.TempDir()
			baseDirectory := filepath.Join(root, "base")
			testutil.RequireMkdirAll(t, baseDirectory)
			testutil.RequireWriteComposeFile(t, baseDirectory, `services:
  common:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_EXTENDS_GREETING:?Greeting is required}"}
`)
			testutil.RequireWriteFile(t, filepath.Join(baseDirectory, ".env"), "TOPO_TEST_EXTENDS_GREETING=ignored\n")
			path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    extends: {file: ./base/compose.yaml, service: common}
x-topo:
  parameters: {TOPO_TEST_EXTENDS_GREETING: {}}
`)
			if scenario.value != "" {
				t.Setenv("TOPO_TEST_EXTENDS_GREETING", scenario.value)
			}
			resolver := parameter.NewStrictResolverChain()

			got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

			if scenario.wantError {
				require.ErrorContains(t, err, "TOPO_TEST_EXTENDS_GREETING")
			} else {
				require.NoError(t, err)
			}
			assert.Empty(t, got)
		})
	}
}
