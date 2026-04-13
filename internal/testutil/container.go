package testutil

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const containerHost = "root@localhost"

type Container struct {
	SSHDestination string
	ContainerName  string
}

type containerSpec struct {
	dockerfileDir string
	image         string
	runArgs       []string
	waitFunc      func(t *testing.T, host, port string)
}

var dindSpec = containerSpec{
	dockerfileDir: "test-container",
	image:         "topo-e2e-target:latest",
	runArgs:       []string{"--privileged"},
	waitFunc:      waitForDockerReady,
}

var sshSpec = containerSpec{
	dockerfileDir: "ssh-container",
	image:         "topo-e2e-ssh:latest",
	waitFunc:      waitForSSHReady,
}

func StartDinDContainer(t *testing.T) *Container {
	t.Helper()
	return startContainer(t, dindSpec)
}

func StartSSHContainer(t *testing.T) *Container {
	t.Helper()
	return startContainer(t, sshSpec)
}

func startContainer(t *testing.T, spec containerSpec) *Container {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that requires a container in short mode")
	}
	RequireDockerEngine(t)

	buildImage(t, spec)

	containerName := generateContainerName(t)
	t.Cleanup(func() {
		deleteContainer(containerName)
	})

	if err := runContainer(containerName, spec); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	port, err := GetContainerPublicPort(containerName, "22")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	spec.waitFunc(t, containerHost, port)

	return &Container{
		SSHDestination: fmt.Sprintf("ssh://%s:%s", containerHost, port),
		ContainerName:  containerName,
	}
}

func buildImage(t *testing.T, spec containerSpec) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	contextDir := filepath.Join(filepath.Dir(thisFile), spec.dockerfileDir)
	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("docker", "build", "-t", spec.image, contextDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build image %s: %v", spec.image, err)
	}
}

func generateContainerName(t *testing.T) string {
	return fmt.Sprintf("topo-test-%s", SanitiseTestName(t))
}

func runContainer(containerName string, spec containerSpec) error {
	deleteContainer(containerName)
	// #nosec G204 -- ignore as its a test helper
	args := append([]string{"run", "--name", containerName, "--detach", "-P"}, spec.runArgs...)
	args = append(args, spec.image)
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
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

func waitForSSHReady(t *testing.T, host string, port string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		// #nosec G204 -- ignore as its a test helper
		cmd := exec.Command("ssh", "-p", port, "-o", "ConnectTimeout=2", "-o", "StrictHostKeyChecking=accept-new", "--", host, "echo", "ready")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return
		}
		lastErr = fmt.Errorf("ssh check failed: %w output: %s", err, strings.TrimSpace(string(output)))
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("ssh not ready in container: %v", lastErr)
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
