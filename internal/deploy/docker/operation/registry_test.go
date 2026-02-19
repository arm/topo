package operation_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
	op "github.com/arm/topo/internal/deploy/operation"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunRegistry(t *testing.T) {
	t.Run("returns expected sequence", func(t *testing.T) {
		port := operation.DefaultRegistryPort

		got := operation.NewRunRegistry(port)

		want := op.NewSequence(
			operation.NewDockerPull(ssh.PlainLocalhost, "registry:2"),
			op.NewConditional(
				operation.NewContainerExistsPredicate(ssh.PlainLocalhost, operation.RegistryContainerName),
				operation.NewDockerStart(ssh.PlainLocalhost, operation.RegistryContainerName),
				operation.NewRegistryRunWrapper(operation.NewDockerRun(ssh.PlainLocalhost, "registry:2", operation.RegistryContainerName,
					[]string{
						"-d",
						"--restart", "always",
						"-p", fmt.Sprintf("127.0.0.1:%s:5000", port),
					},
				)),
			),
		)
		assert.Equal(t, want, got)
	})
}

func TestContainerExistsPredicate(t *testing.T) {
	t.Run("evaluates to true when container exists", func(t *testing.T) {
		testutil.RequireLinuxDockerEngine(t)
		containerName := testutil.TestContainerName(t)
		imageName := testutil.TestImageName(t)
		testutil.BuildMinimalImage(t, ssh.PlainLocalhost, imageName)
		runCmd := command.Docker(ssh.PlainLocalhost, "run", "-d", "--name", containerName, imageName)
		require.NoError(t, runCmd.Run())
		t.Cleanup(func() {
			stopCmd := command.Docker(ssh.PlainLocalhost, "rm", "-f", containerName)
			_ = stopCmd.Run()
		})

		predicate := operation.NewContainerExistsPredicate(ssh.PlainLocalhost, containerName)
		got := predicate.Eval()

		assert.True(t, got)
	})

	t.Run("evaluates to false when container does not exist", func(t *testing.T) {
		testutil.RequireDocker(t)
		containerName := "non-existent-container-12345"

		predicate := operation.NewContainerExistsPredicate(ssh.PlainLocalhost, containerName)
		got := predicate.Eval()

		assert.False(t, got)
	})
}
