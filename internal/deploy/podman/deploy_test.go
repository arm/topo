package podman_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDeploy(t *testing.T) {
	requireLocalPodman(t)

	t.Run("deploys to localhost", func(t *testing.T) {
		composeFile, projectName := deploymentFixture(t)
		t.Cleanup(func() { cleanupComposeProject(t, composeFile) })
		options := podman.DeployOptions{TargetHost: ssh.PlainLocalhost}

		err := podman.Deploy(t.Context(), t.Output(), composeFile, options)

		require.NoError(t, err)
		assertContainersRunning(t, projectName, podman.LocalSocket)
	})

	t.Run("transfers images to a remote host via pipe", func(t *testing.T) {
		target := startContainer(t)
		composeFile, projectName := deploymentFixture(t)
		targetDestination := ssh.NewDestination(target.SSHDestination)
		options := podman.DeployOptions{TargetHost: targetDestination}

		err := podman.Deploy(t.Context(), t.Output(), composeFile, options)

		require.NoError(t, err)
		tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetDestination)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tunnel.Close())
		})
		assertContainersRunning(t, projectName, podman.NewSocket(tunnel.SocketURL()))
	})

	t.Run("transfers images to a remote host through a registry", func(t *testing.T) {
		registryPort := requireAvailableTCPPort(t)
		registryContainerName := "topo-test-registry-" + sanitiseTestName(t)
		cleanupRegistryContainer(t, registryContainerName)
		target := startContainer(t)
		composeFile, projectName := deploymentFixture(t)
		targetDestination := ssh.NewDestination(target.SSHDestination)
		options := podman.DeployOptions{
			TargetHost: targetDestination,
			Registry: &podman.RegistryConfig{
				ContainerName: registryContainerName,
				Port:          registryPort,
			},
		}

		err := podman.Deploy(t.Context(), t.Output(), composeFile, options)

		require.NoError(t, err)
		tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetDestination)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tunnel.Close())
		})
		assertContainersRunning(t, projectName, podman.NewSocket(tunnel.SocketURL()))
	})
}

func requireLocalPodman(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman is not installed")
	}
	if _, err := exec.LookPath("docker-compose"); err != nil {
		t.Skip("docker-compose is not installed")
	}
	if output, err := exec.Command("podman", "info").CombinedOutput(); err != nil {
		t.Skipf("local Podman engine is unavailable: %v: %s", err, output)
	}
}

func assertContainersRunning(t *testing.T, projectName string, socket podman.Socket) {
	t.Helper()
	cmd := podman.Command(t.Context(), socket,
		"ps", "--format", "json", "--all",
		"--filter", "label=com.docker.compose.project="+projectName,
	)
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	output, err := cmd.Output()
	require.NoError(t, err, "stdout: %s\nstderr: %s", output, diagnostics.String())

	var containers []map[string]any
	require.NoError(t, json.Unmarshal(output, &containers))
	require.NotEmpty(t, containers, "no containers reported; stderr: %s", diagnostics.String())

	for _, container := range containers {
		assert.Equal(t, "running", container["State"], "container %s is not running: %s", container["Names"], container["State"])
	}
}

func deploymentFixture(t *testing.T) (string, string) {
	t.Helper()
	tempDir := t.TempDir()
	testName := sanitiseTestName(t)
	imageName := "test-image-" + testName
	composeFile := filepath.Join(tempDir, "compose.yaml")
	composeFileContent := fmt.Sprintf(`
name: %s
services:
  built:
    build: .
    image: %s
  pulled:
    image: docker.io/library/alpine:latest
    command: ["tail", "-f", "/dev/null"]
`, "test-project-"+testName, imageName)
	composeFileContent, err := fixPodmanInDockerQuirk(composeFileContent)
	require.NoError(t, err)
	requireWriteFile(t, composeFile, composeFileContent)
	requireWriteFile(t, filepath.Join(tempDir, "Dockerfile"), `
FROM docker.io/library/alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		removeOutput, err := podman.Command(ctx, podman.LocalSocket, "image", "rm", "-f", imageName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove image %s: %v: %s", imageName, err, string(removeOutput))
		}
	})
	return composeFile, "test-project-" + testName
}

// fixPodmanInDockerQuirk avoids a Docker Desktop nested-container restriction.
// The Podman target inherits oom_score_adj: 200, but Podman otherwise starts
// each service with oom_score_adj: 0. Docker Desktop rejects that decrease, so
// this adds oom_score_adj: 200 to each fixture service, for example:
//
//	services:
//	  app:
//	    oom_score_adj: 200
func fixPodmanInDockerQuirk(contents string) (string, error) {
	var definition map[string]any
	if err := yaml.Unmarshal([]byte(contents), &definition); err != nil {
		return "", err
	}
	services, ok := definition["services"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("Compose file services must be a mapping")
	}
	for name, value := range services {
		service, ok := value.(map[string]any)
		if !ok {
			return "", fmt.Errorf("service %q must be a mapping", name)
		}
		service["oom_score_adj"] = 200
	}

	updatedContents, err := yaml.Marshal(definition)
	if err != nil {
		return "", err
	}
	return string(updatedContents), nil
}

func cleanupRegistryContainer(t *testing.T, containerName string) {
	t.Helper()
	_ = podman.Command(t.Context(), podman.LocalSocket, "rm", "-f", containerName).Run()
	t.Cleanup(func() {
		output, err := podman.Command(context.Background(), podman.LocalSocket, "rm", "-f", containerName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove registry container: %v: %s", err, output)
		}
	})
}

func cleanupComposeProject(t *testing.T, composeFile string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd, err := podman.ComposeCommand(ctx, podman.LocalSocket, composeFile, "down", "-v", "--remove-orphans", "--rmi", "local")
	if err != nil {
		t.Logf("failed to configure Podman Compose: %v", err)
		return
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("Podman Compose cleanup failed: %v: %s", err, output)
	}
}
