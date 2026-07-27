package docker_test

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestStop(t *testing.T) {
	testutil.RequireDocker(t)

	container := testutil.StartContainer(t, testutil.DinDContainer)
	remoteDockerHost := ssh.NewDestination(container.SSHDestination)
	temporaryDirectory := t.TempDir()
	dockerFilePath := filepath.Join(temporaryDirectory, "Dockerfile")
	testutil.RequireWriteFile(t, dockerFilePath, `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	composeFilePath := filepath.Join(temporaryDirectory, "compose.yaml")
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
	deployOptions := deploy.DeployOptions{TargetHost: remoteDockerHost}
	require.NoError(t, deploy.Deploy(t.Context(), io.Discard, composeFilePath, deployOptions))
	testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)

	err := docker.Stop(t.Context(), io.Discard, composeFilePath, remoteDockerHost)

	require.NoError(t, err)
	testutil.AssertContainersStopped(t, remoteDockerHost, composeFilePath)
}
