package deploy_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/operation"
	"github.com/arm/topo/internal/deploy/testutil"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeploymentStop(t *testing.T) {
	composeFile := "compose.yaml"

	t.Run("runs stop operation for remote host", func(t *testing.T) {
		remoteDest := ssh.NewDestination("user@remote")
		remoteHost := command.NewHostFromDestination(remoteDest)

		got := deploy.NewDeploymentStop(composeFile, remoteDest)

		want := goperation.Sequence{
			operation.NewDockerComposeStop(composeFile, remoteHost),
		}
		assert.Equal(t, want, got)
	})

	t.Run("runs stop operation for local host", func(t *testing.T) {
		got := deploy.NewDeploymentStop(composeFile, ssh.PlainLocalhost)

		want := goperation.Sequence{
			operation.NewDockerComposeStop(composeFile, command.LocalHost),
		}
		assert.Equal(t, want, got)
	})
}

func TestDeploymentStop(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Run", func(t *testing.T) {
		container := testutil.StartContainer(t, testutil.DinDContainer)

		t.Run("deploys services then confirms stop shuts down containers", func(t *testing.T) {
			remoteDockerHost := ssh.NewDestination(container.SSHDestination)
			tmpDir := t.TempDir()
			dockerFilePath := filepath.Join(tmpDir, "Dockerfile")
			dockerFileContent := `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`
			testutil.RequireWriteFile(t, dockerFilePath, dockerFileContent)
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := fmt.Sprintf(`
name: %s
services:
  busybox:
    image: busybox
    command: ["tail", "-f", "/dev/null"]
  a-service:
    build: .
`, testutil.TestProjectName(t))
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			t.Cleanup(func() { testutil.ForceComposeDown(t, composeFilePath) })

			deployOpts := deploy.DeployOptions{TargetHost: remoteDockerHost}
			deployment, _ := deploy.NewDeployment(composeFilePath, deployOpts)

			require.NoError(t, deployment.Run(os.Stdout))
			testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)

			stop := deploy.NewDeploymentStop(composeFilePath, remoteDockerHost)
			err := stop.Run(os.Stdout)

			require.NoError(t, err)
			testutil.AssertContainersStopped(t, remoteDockerHost, composeFilePath)
		})
	})
}
