package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/e2e/testutil"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestDeploymentStop(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		target := testutil.StartTargetContainer(t)

		t.Run("deploys services then confirms stop shuts down containers", func(t *testing.T) {
			remoteDockerHost := ssh.Host(target.SSHDestination)
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

			deployOpts := docker.DeployOptions{TargetHost: remoteDockerHost}
			deploy, _ := docker.NewDeployment(composeFilePath, deployOpts)

			require.NoError(t, deploy.Run(os.Stdout))
			testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)

			stop := docker.NewDeploymentStop(composeFilePath, remoteDockerHost)
			err := stop.Run(os.Stdout)

			require.NoError(t, err)
			testutil.AssertContainersStopped(t, remoteDockerHost, composeFilePath)
		})
	})
}
