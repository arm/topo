package docker_test

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestStop(t *testing.T) {
	requireDocker(t)

	container := startContainer(t, dinDContainer)
	remoteDockerHost := ssh.NewDestination(container.SSHDestination)
	temporaryDirectory := t.TempDir()
	dockerFilePath := filepath.Join(temporaryDirectory, "Dockerfile")
	requireWriteFile(t, dockerFilePath, `
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
`, testProjectName(t))
	requireWriteFile(t, composeFilePath, composeFileContent)
	t.Cleanup(func() { forceComposeDown(t, composeFilePath) })
	deployOptions := docker.DeployOptions{TargetHost: remoteDockerHost}
	require.NoError(t, docker.Deploy(t.Context(), io.Discard, composeFilePath, deployOptions))
	assertContainersRunning(t, remoteDockerHost, composeFilePath)

	err := docker.Stop(t.Context(), io.Discard, composeFilePath, remoteDockerHost)

	require.NoError(t, err)
	assertContainersStopped(t, remoteDockerHost, composeFilePath)
}
