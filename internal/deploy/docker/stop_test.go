package docker_test

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestStop(t *testing.T) {
	requireDocker(t)

	container := startContainer(t, dinDContainer)
	remoteDockerHost := ssh.NewDestination(container.SSHDestination)
	temporaryDirectory := t.TempDir()
	dockerFilePath := filepath.Join(temporaryDirectory, "Dockerfile")
	testutil.RequireWriteFile(t, dockerFilePath, `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	composeFilePath := testutil.RequireWriteComposeFile(t, temporaryDirectory, fmt.Sprintf(`
name: %s
services:
  busybox:
    image: busybox
    command: ["tail", "-f", "/dev/null"]
    stop_grace_period: 1s
  a-service:
    build: .
    stop_grace_period: 1s
`, testProjectName(t)))
	scope := project.Scope{ComposeFile: composeFilePath}
	t.Cleanup(func() { forceComposeDown(t, scope) })
	deployOptions := docker.DeployOptions{TargetHost: remoteDockerHost}
	require.NoError(t, docker.Deploy(t.Context(), io.Discard, scope, deployOptions))
	assertContainersRunning(t, remoteDockerHost, scope)

	err := docker.Stop(t.Context(), io.Discard, scope, remoteDockerHost)

	require.NoError(t, err)
	assertContainersStopped(t, remoteDockerHost, scope)
}
