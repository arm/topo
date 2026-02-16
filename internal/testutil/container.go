package testutil

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const TargetContainerHost = "root@localhost"

const TargetContainerImage = "topo-e2e-target:latest"

var knownHostsLockPath = filepath.Join(os.TempDir(), "topo-e2e-known_hosts.lock")

type TargetContainer struct {
	SSHConnectionString string
	ContainerName       string
}

func StartTargetContainer(t *testing.T) *TargetContainer {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that requires a target container in short mode")
	}
	RequireLinuxDockerEngine(t)

	containerName := generateTargetContainerName(t)

	t.Cleanup(func() {
		deleteContainer(containerName)
	})

	if err := createTargetContainer(t, containerName); err != nil {
		t.Fatalf("failed to create vm: %v", err)
	}

	port, err := GetContainerPublicPort(containerName, "22")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	if err := ensureHostKeyKnown(t, "localhost", port); err != nil {
		t.Fatalf("failed to add host key: %v", err)
	}

	waitForDockerReady(t, TargetContainerHost, port)

	return &TargetContainer{SSHConnectionString: fmt.Sprintf("%s:%s", TargetContainerHost, port), ContainerName: containerName}
}

func generateTargetContainerName(t *testing.T) string {
	return fmt.Sprintf("topo-test-%s", SanitiseTestName(t))
}

func requireImageExists(t *testing.T, imageName string) {
	t.Helper()
	cmd := exec.Command("docker", "images", "-q", imageName)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to check for docker image %s: %v", imageName, err)
	}
	if len(strings.TrimSpace(string(output))) == 0 {
		t.Skipf("required docker image %s not found. Please build it before running the tests.", imageName)
	}
}

func createTargetContainer(t *testing.T, containerName string) error {
	t.Helper()
	requireImageExists(t, TargetContainerImage)
	deleteContainer(containerName)
	cmd := exec.Command("docker", "run", "--name", containerName, "--detach", "-P", "--privileged", TargetContainerImage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start target container: %w", err)
	}
	return nil
}

func deleteContainer(containerName string) {
	cmd := exec.Command("docker", "rm", "--force", containerName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func GetContainerPublicPort(containerName string, privatePort string) (string, error) {
	cmd := exec.Command("docker", "port", containerName, privatePort)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get container port: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no port mapping found")
	}
	_, port, err := net.SplitHostPort(lines[0])
	if err != nil {
		return "", fmt.Errorf("failed to parse port mapping: %w", err)
	}
	return port, nil
}

func ensureHostKeyKnown(t *testing.T, host string, port string) error {
	releaseLock := AcquireFlock(t, knownHostsLockPath)
	defer releaseLock()

	_ = exec.Command("ssh-keygen", "-R", fmt.Sprintf("[%s]:%s", host, port)).Run()

	cmd := exec.Command("ssh-keyscan", "-p", port, host)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ssh-keyscan failed: %w", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("ssh-keyscan returned no host keys")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")
	if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0o700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(output); err != nil {
		return fmt.Errorf("failed to write to known_hosts: %w", err)
	}

	return nil
}

func waitForDockerReady(t *testing.T, host string, port string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		cmd := exec.Command("ssh", "-p", port, "-o", "ConnectTimeout=2", host, "docker", "info")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return
		}
		lastErr = fmt.Errorf("docker info failed: %w output: %s", err, strings.TrimSpace(string(output)))
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("docker daemon not ready in target container: %v", lastErr)
}
