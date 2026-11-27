package checks_test

import (
	"testing"

	checks "github.com/arm/topo/internal/deploy/project_checks"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestEnsureProjectIsLinuxArm64Ready(t *testing.T) {
	t.Run("succeeds with valid platforms without variant", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  app:
    image: alpine
    platform: linux/arm64
`)

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := checks.EnsureProjectIsLinuxArm64Ready(scope)

		require.NoError(t, err)
	})

	t.Run("succeeds with valid platforms with variant", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  app:
    image: alpine
    platform: linux/arm64/v8
`)

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := checks.EnsureProjectIsLinuxArm64Ready(scope)

		require.NoError(t, err)
	})
	t.Run("fails when platform missing", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  api:
    image: busybox
`)

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := checks.EnsureProjectIsLinuxArm64Ready(scope)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing platform declaration")
	})
	t.Run("fails when platform is not linux/arm64", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  api:
    image: busybox
    platform: linux/amd64
`)

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := checks.EnsureProjectIsLinuxArm64Ready(scope)
		require.Error(t, err)
		require.Contains(t, err.Error(), "linux/amd64")
	})
	t.Run("skips remoteproc-runtime without platform", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  firmware:
    image: zephyr
    runtime: io.containerd.remoteproc.v1
`)

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := checks.EnsureProjectIsLinuxArm64Ready(scope)

		require.NoError(t, err)
	})
	t.Run("succeeds with valid remoteproc-runtime", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  rtos-firmware:
    build:
      context: .
    runtime: io.containerd.remoteproc.v1
`)

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := checks.EnsureProjectIsLinuxArm64Ready(scope)

		require.NoError(t, err)
	})
}
