package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const TargetContainerHost = "root@localhost"

type TargetContainer struct {
	SSHDestination string
	container      testcontainers.Container
}

func StartTargetContainer(t *testing.T) *TargetContainer {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that requires a target container in short mode")
	}
	RequireLinuxDockerEngine(t)
	setDockerHostFromContext(t)

	ctx := context.Background()
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:       dockerfileContextPath(),
				Dockerfile:    "Dockerfile",
				PrintBuildLog: true,
			},
			ExposedPorts: []string{"22/tcp", "8080/tcp"},
			Privileged:   true,
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("22/tcp"),
				wait.ForExec([]string{"docker", "info"}).
					WithPollInterval(200*time.Millisecond).
					WithStartupTimeout(20*time.Second),
			),
		},
		Started: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, ctr.Terminate(context.Background()))
	})

	sshPort, err := ctr.MappedPort(ctx, "22/tcp")
	require.NoError(t, err)

	acceptSSHHostKey(t, sshPort.Port())

	return &TargetContainer{
		SSHDestination: fmt.Sprintf("ssh://%s:%s", TargetContainerHost, sshPort.Port()),
		container:      ctr,
	}
}

func (tc *TargetContainer) MappedPort(t *testing.T, port string) string {
	t.Helper()
	p, err := tc.container.MappedPort(context.Background(), port)
	require.NoError(t, err)
	return p.Port()
}

func acceptSSHHostKey(t *testing.T, port string) {
	t.Helper()
	// #nosec G204 -- test helper
	cmd := exec.Command("ssh", "-p", port, "-o", "ConnectTimeout=5", "-o", "StrictHostKeyChecking=accept-new", "--", TargetContainerHost, "true")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to accept SSH host key: %s", string(out))
}

func dockerfileContextPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "test-container")
}

// setDockerHostFromContext sets DOCKER_HOST from the active Docker context
// when not already set. testcontainers-go doesn't read Docker contexts.
// It also sets TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE for VM-based Docker
// setups (e.g. Lima) where the host socket path doesn't exist inside the VM.
func setDockerHostFromContext(t *testing.T) {
	t.Helper()
	if os.Getenv("DOCKER_HOST") != "" {
		return
	}
	// #nosec G204 -- test helper
	out, err := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return
	}
	host := strings.TrimSpace(string(out))
	if host == "" {
		return
	}
	t.Setenv("DOCKER_HOST", host)

	if os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE") == "" {
		t.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
	}
	// Ryuk often fails in VM-based Docker setups (e.g. Lima).
	// t.Cleanup handles container removal, so Ryuk is not essential.
	if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "" {
		t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	}
}
