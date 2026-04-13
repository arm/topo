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
	postReady     func(t *testing.T, containerName string)
}

var dindSpec = containerSpec{
	dockerfileDir: "test-container",
	image:         "topo-e2e-target:latest",
	runArgs:       []string{"--privileged"},
	postReady:     waitForDockerDaemon,
}

var sshSpec = containerSpec{
	dockerfileDir: "ssh-container",
	image:         "topo-e2e-ssh:latest",
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

	waitForPort(t, "localhost", port, 10*time.Second)

	c := &Container{
		SSHDestination: fmt.Sprintf("ssh://%s:%s", containerHost, port),
		ContainerName:  containerName,
	}

	acceptHostKey(t, c, port)

	if spec.postReady != nil {
		spec.postReady(t, containerName)
	}

	return c
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

func acceptHostKey(t *testing.T, c *Container, port string) {
	t.Helper()
	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("ssh", c.SSHDestination, "-o", "StrictHostKeyChecking=accept-new", "true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to accept host key: %v output: %s", err, strings.TrimSpace(string(output)))
	}
	t.Cleanup(func() {
		// #nosec G204 -- ignore as its a test helper
		_ = exec.Command("ssh-keygen", "-R", "[localhost]:"+port).Run()
	})
}

func AcceptHostKeyFor(t *testing.T, c *Container, sshDir string) {
	t.Helper()
	knownHostsPath := filepath.Join(sshDir, "known_hosts")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("failed to create ssh dir: %v", err)
	}
	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("ssh", c.SSHDestination, "-o", "StrictHostKeyChecking=accept-new", "-o", "UserKnownHostsFile="+knownHostsPath, "true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to accept host key: %v output: %s", err, strings.TrimSpace(string(output)))
	}
}

func waitForPort(t *testing.T, host string, port string, timeout time.Duration) {
	t.Helper()
	addr := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("port %s not ready: %v", addr, lastErr)
}

func waitForDockerDaemon(t *testing.T, containerName string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		// #nosec G204 -- ignore as its a test helper
		cmd := exec.Command("docker", "exec", containerName, "docker", "info")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return
		}
		lastErr = fmt.Errorf("docker daemon not ready: %w output: %s", err, strings.TrimSpace(string(output)))
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("docker daemon not ready in container: %v", lastErr)
}
