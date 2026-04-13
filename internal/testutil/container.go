package testutil

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type Container struct {
	SSHDestination string
	ContainerName  string
}

type ContainerSpec struct {
	context string
	image   string
	runArgs []string
	setup   func(c *Container) error
	cleanup func(c *Container)
}

var DinDContainer = ContainerSpec{
	context: relPath("test-container"),
	image:   "topo-e2e-target:latest",
	runArgs: []string{"--privileged"},
	setup: func(c *Container) error {
		if err := acceptHostKey(c); err != nil {
			return err
		}
		return waitForDockerDaemon(c)
	},
	cleanup: func(c *Container) {
		removeHostKey(c)
	},
}

func StartContainer(t *testing.T, spec ContainerSpec) *Container {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that requires a container in short mode")
	}
	RequireLinuxDockerEngine(t)

	if err := buildImage(spec); err != nil {
		t.Fatalf("failed to build image: %v", err)
	}

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

	if err := waitForPort("localhost", port, 10*time.Second); err != nil {
		t.Fatalf("container port not ready: %v", err)
	}

	c := &Container{
		SSHDestination: fmt.Sprintf("ssh://root@localhost:%s", port),
		ContainerName:  containerName,
	}

	if spec.setup != nil {
		if err := spec.setup(c); err != nil {
			t.Fatalf("container setup failed: %v", err)
		}
	}
	if spec.cleanup != nil {
		t.Cleanup(func() {
			spec.cleanup(c)
		})
	}

	return c
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

func relPath(dir string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), dir)
}

func buildImage(spec ContainerSpec) error {
	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("docker", "build", "-t", spec.image, spec.context)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build %s: %w", spec.image, err)
	}
	return nil
}

func generateContainerName(t *testing.T) string {
	return fmt.Sprintf("topo-test-%s", SanitiseTestName(t))
}

func runContainer(containerName string, spec ContainerSpec) error {
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

func acceptHostKey(c *Container) error {
	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("ssh", c.SSHDestination, "-o", "StrictHostKeyChecking=accept-new", "true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w output: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func removeHostKey(c *Container) {
	u, err := url.Parse(c.SSHDestination)
	if err != nil {
		return
	}
	host := fmt.Sprintf("[%s]:%s", u.Hostname(), u.Port())
	// #nosec G204 -- ignore as its a test helper
	_ = exec.Command("ssh-keygen", "-R", host).Run()
}

func waitForPort(host string, port string, timeout time.Duration) error {
	addr := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("port %s not ready: %w", addr, lastErr)
}

func waitForDockerDaemon(c *Container) error {
	containerName := c.ContainerName
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		// #nosec G204 -- ignore as its a test helper
		cmd := exec.Command("docker", "exec", containerName, "docker", "info")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("%w output: %s", err, strings.TrimSpace(string(output)))
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("docker daemon not ready: %w", lastErr)
}
