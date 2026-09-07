package podman_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestEnsureNoRuntimeSet(t *testing.T) {
	t.Run("succeeds when no service runtime is set", func(t *testing.T) {
		composeFile := testutil.WriteComposeFile(t, t.TempDir(), `
services:
  app:
    image: alpine
`)

		err := podman.EnsureNoRuntimeSet(composeFile)

		require.NoError(t, err)
	})

	t.Run("rejects a service with a runtime", func(t *testing.T) {
		composeFile := testutil.WriteComposeFile(t, t.TempDir(), `
services:
  firmware:
    image: alpine
    runtime: io.containerd.remoteproc.v1
`)

		err := podman.EnsureNoRuntimeSet(composeFile)

		require.EqualError(t, err, `specifying "runtime:" in Compose files is unsupported for Podman deployments: "firmware" service uses "io.containerd.remoteproc.v1"`)
	})

	t.Run("lists every service with a runtime", func(t *testing.T) {
		composeFile := testutil.WriteComposeFile(t, t.TempDir(), `
services:
  application:
    image: alpine
    runtime: kata
  firmware:
    image: alpine
    runtime: io.containerd.remoteproc.v1
`)

		err := podman.EnsureNoRuntimeSet(composeFile)

		require.EqualError(t, err, `specifying "runtime:" in Compose files is unsupported for Podman deployments: "application" service uses "kata", "firmware" service uses "io.containerd.remoteproc.v1"`)
	})
}
