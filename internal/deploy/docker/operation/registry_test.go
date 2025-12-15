package operation_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/testutil"
	op "github.com/arm-debug/topo-cli/internal/deploy/operation"

	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunRegistry(t *testing.T) {
	t.Run("returns expected sequence", func(t *testing.T) {
		host := ssh.Host("user@remote")

		got := operation.NewRunRegistry(host)

		want := op.NewSequence(
			operation.NewPull(ssh.PlainLocalhost, "registry:2"),
			operation.NewPipeTransfer("registry:2", ssh.PlainLocalhost, host),
			operation.NewStartOrRun(host, operation.RegistryContainerName, "registry:2",
				"-d", "--restart=always", fmt.Sprintf("-p=127.0.0.1:%d:5000", operation.RegistryPort)),
		)
		assert.Equal(t, want, got)
	})
}

func TestPull(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		t.Run("returns image name", func(t *testing.T) {
			pull := operation.NewPull(ssh.PlainLocalhost, "registry:2")

			assert.Equal(t, "Pull image registry:2", pull.Description())
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints docker pull command", func(t *testing.T) {
			var buf bytes.Buffer
			pull := operation.NewPull(ssh.PlainLocalhost, "registry:2")

			require.NoError(t, pull.DryRun(&buf))

			assert.Equal(t, "docker pull registry:2\n", buf.String())
		})
	})
}

func TestPipeTransfer(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		t.Run("returns image name", func(t *testing.T) {
			transfer := operation.NewPipeTransfer("registry:2", ssh.PlainLocalhost, ssh.Host("user@remote"))

			assert.Equal(t, "Transfer image registry:2", transfer.Description())
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints save and load commands", func(t *testing.T) {
			var buf bytes.Buffer
			transfer := operation.NewPipeTransfer("registry:2", ssh.PlainLocalhost, ssh.Host("user@remote"))

			require.NoError(t, transfer.DryRun(&buf))

			expected := "docker save registry:2 | docker -H ssh://user@remote load\n"
			assert.Equal(t, expected, buf.String())
		})
	})
}

func TestStartOrRun(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		t.Run("returns container name", func(t *testing.T) {
			startOrRun := operation.NewStartOrRun(ssh.Host("user@remote"), "my-container", "my-image:latest")

			assert.Equal(t, "Start image my-container", startOrRun.Description())
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		testutil.RequireDocker(t)

		t.Run("prints run command when container does not exist", func(t *testing.T) {
			var buf bytes.Buffer
			startOrRun := operation.NewStartOrRun(ssh.Host("user@remote"), "topo-registry", "registry:2",
				"-d", "--restart=always", fmt.Sprintf("-p=127.0.0.1:%d:5000", operation.RegistryPort))

			require.NoError(t, startOrRun.DryRun(&buf))

			expected := "docker -H ssh://user@remote run -d --restart=always -p=127.0.0.1:12737:5000 --name=topo-registry registry:2\n"
			assert.Equal(t, expected, buf.String())
		})
	})
}
