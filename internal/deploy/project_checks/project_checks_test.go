package checks_test

import (
	"testing"

	checks "github.com/arm/topo/internal/deploy/project_checks"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestEnsureProjectIsLinuxArm64Ready(t *testing.T) {
	t.Run("succeeds with valid platforms without variant", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  app:
    image: alpine
    platform: linux/arm64
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})

	t.Run("succeeds with valid platforms with variant", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  app:
    image: alpine
    platform: linux/arm64/v8
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})
	t.Run("fails when platform missing", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  api:
    image: busybox
`)

		err := checks.EnsureProjectIsLinuxArm64Ready(composeFile)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing platform declaration")
	})
	t.Run("fails when platform is not linux/arm64", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  api:
    image: busybox
    platform: linux/amd64
`)

		err := checks.EnsureProjectIsLinuxArm64Ready(composeFile)
		require.Error(t, err)
		require.Contains(t, err.Error(), "linux/amd64")
	})
	t.Run("skips remoteproc runtime without platform", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  firmware:
    image: zephyr
    runtime: io.containerd.remoteproc.v1
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})
	t.Run("succeeds with valid remoteproc runtime", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  rtos-firmware:
    build:
      context: .
    runtime: io.containerd.remoteproc.v1
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})
}

func writeComposeFile(t *testing.T, contents string) string {
	t.Helper()
	return testutil.WriteComposeFile(t, t.TempDir(), contents)
}
