package testutil

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
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
	Name           string
}

type ContainerSpec struct {
	context string
	image   string
	runArgs []string
	setup   func(c *Container) error
}

var PasswordedSSHContainer = ContainerSpec{
	context: relPath("passworded-ssh"),
	image:   "topo-e2e-passworded-ssh:latest",
}

var DinDContainer = ContainerSpec{
	context: relPath("dind"),
	image:   "topo-e2e-dind:latest",
	runArgs: []string{"--privileged"},
	setup: func(c *Container) error {
		if err := acceptHostKey(c); err != nil {
			return err
		}
		return waitForDockerDaemon(c)
	},
}

var PasswordlessSSHContainer = ContainerSpec{
	context: relPath("passwordless-ssh"),
	image:   "topo-e2e-passwordless-ssh:latest",
	setup: func(c *Container) error {
		if err := acceptHostKey(c); err != nil {
			return err
		}
		return nil
	},
}

var PodmanContainer = ContainerSpec{
	context: relPath("podman"),
	image:   "topo-e2e-podman:latest",
	runArgs: []string{"--privileged"},
	setup: func(c *Container) error {
		if err := acceptHostKey(c); err != nil {
			return err
		}
		return waitForPodmanService(c)
	},
}

func StartContainer(t *testing.T, spec ContainerSpec) *Container {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that requires a container in short mode")
	}
	RequireLinuxDockerEngine(t)
	containerName := generateContainerName(t)
	requireKnownHostsSSHConfig(t, containerName)

	if err := buildImage(spec); err != nil {
		t.Fatalf("failed to build image: %v", err)
	}

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

	t.Cleanup(func() { removeKnownHost(t, containerName) })

	if err := waitForSSH("localhost", port, 10*time.Second); err != nil {
		t.Fatalf("container SSH not ready: %v", err)
	}

	c := &Container{
		SSHDestination: fmt.Sprintf("ssh://root@%s:%s", containerName, port),
		Name:           containerName,
	}

	if spec.setup != nil {
		if err := spec.setup(c); err != nil {
			t.Fatalf("container setup failed: %v", err)
		}
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
	return fmt.Sprintf("topo-test-%s-%s", SanitiseTestName(t), strings.ToLower(rand.Text()))
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

func waitForSSH(host string, port string, timeout time.Duration) error {
	addr := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	lastErr := os.ErrDeadlineExceeded

	for time.Now().Before(deadline) {
		lastErr = readSSHBanner(addr, min(2*time.Second, time.Until(deadline)))
		if lastErr == nil {
			return nil
		}
		time.Sleep(min(200*time.Millisecond, time.Until(deadline)))
	}

	return fmt.Errorf("SSH at %s not ready: %w", addr, lastErr)
}

func readSSHBanner(addr string, timeout time.Duration) (err error) {
	deadline := time.Now().Add(timeout)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	if err := conn.SetReadDeadline(deadline); err != nil {
		return err
	}
	banner, err := bufio.NewReader(io.LimitReader(conn, 255)).ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(banner, "SSH-") {
		return fmt.Errorf("unexpected SSH banner: %q", banner)
	}
	return nil
}

func waitForPodmanService(c *Container) error {
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		// #nosec G204 -- ignore as its a test helper
		cmd := exec.Command("docker", "exec", c.Name, "podman", "--url", "unix:///run/podman/podman.sock", "info")
		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("%w output: %s", err, strings.TrimSpace(string(output)))
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("podman service not ready: %w", lastErr)
}

func waitForDockerDaemon(c *Container) error {
	containerName := c.Name
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
