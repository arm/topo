package checks_test

import (
	"os"
	"path/filepath"
	"testing"

	checks "github.com/arm/topo/internal/deploy/project_checks"
	"github.com/stretchr/testify/require"
)

func TestEnsureProjectIsLinuxArm64Ready_SucceedsWithValidRemoteProc(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  rtos-firmware:
    build:
      context: .
    runtime: io.containerd.remoteproc.v1
`)

	require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
}

func TestEnsureProjectIsLinuxArm64Ready_SucceedsWithValidPlatforms(t *testing.T) {
	t.Run("without variant", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  app:
    image: alpine
    platform: linux/arm64
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})

	t.Run("with variant", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  app:
    image: alpine
    platform: linux/arm64/v8
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})
}

func TestEnsureProjectIsLinuxArm64Ready_FailsWhenPlatformMissing(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  api:
    image: busybox
`)

	err := checks.EnsureProjectIsLinuxArm64Ready(composeFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing platform declaration")
}

func TestEnsureProjectIsLinuxArm64Ready_FailsWhenPlatformMismatch(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  api:
    image: busybox
    platform: linux/amd64
`)

	err := checks.EnsureProjectIsLinuxArm64Ready(composeFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "linux/amd64")
}

func TestEnsureProjectIsLinuxArm64Ready_SkipsRemoteprocRuntime(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  firmware:
    image: zephyr
    runtime: io.containerd.remoteproc.v1
`)

	require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
}

func writeComposeFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}
	return path
}
