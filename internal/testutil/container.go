package testutil

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const TargetContainerHost = "root@localhost"

const TargetContainerImage = "topo-e2e-target:latest"

type TargetContainer struct {
	SSHDestination string
	ContainerName  string
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

	waitForDockerReady(t, TargetContainerHost, port)

	// #nosec G204 -- ignore as its a test helper
	return &TargetContainer{
		SSHDestination: fmt.Sprintf("ssh://%s:%s", TargetContainerHost, port),
		ContainerName:  containerName,
	}
}

func generateTargetContainerName(t *testing.T) string {
	return fmt.Sprintf("topo-test-%s", SanitiseTestName(t))
}

func requireImageExists(t *testing.T, imageName string) {
	t.Helper()
	// #nosec G204 -- ignore as its a test helper
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
	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("docker", "run", "--name", containerName, "--detach", "-P", "--privileged", TargetContainerImage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start target container: %w", err)
	}
	return nil
}

func deleteContainer(containerName string) {
	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("docker", "rm", "--force", containerName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func GetContainerPublicPort(containerName string, privatePort string) (string, error) {
	// #nosec G204 -- ignore as its a test helper
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

func waitForDockerReady(t *testing.T, host string, port string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		// #nosec G204 -- ignore as its a test helper
		cmd := exec.Command("ssh", "-p", port, "-o", "ConnectTimeout=2", "-o", "StrictHostKeyChecking=accept-new", "--", host, "docker", "info")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return
		}
		lastErr = fmt.Errorf("docker info failed: %w output: %s", err, strings.TrimSpace(string(output)))
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("docker daemon not ready in target container: %v", lastErr)
}
