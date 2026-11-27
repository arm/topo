package podman_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestEnsureNoRuntimeSet(t *testing.T) {
	t.Run("succeeds when no service runtime is set", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  app:
    image: alpine
`)

		preparedProject, prepareErr := project.PrepareProject(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := podman.EnsureNoRuntimeSet(preparedProject)

		require.NoError(t, err)
	})

	t.Run("rejects a service with a runtime", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  firmware:
    image: alpine
    runtime: io.containerd.remoteproc.v1
`)

		preparedProject, prepareErr := project.PrepareProject(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := podman.EnsureNoRuntimeSet(preparedProject)

		require.EqualError(t, err, `specifying "runtime:" in Compose files is unsupported for Podman deployments: "firmware" service uses "io.containerd.remoteproc.v1"`)
	})

	t.Run("lists every service with a runtime", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  application:
    image: alpine
    runtime: kata
  firmware:
    image: alpine
    runtime: io.containerd.remoteproc.v1
`)

		preparedProject, prepareErr := project.PrepareProject(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := podman.EnsureNoRuntimeSet(preparedProject)

		require.EqualError(t, err, `specifying "runtime:" in Compose files is unsupported for Podman deployments: "application" service uses "kata", "firmware" service uses "io.containerd.remoteproc.v1"`)
	})
}
