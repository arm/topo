package testutil

import (
	"context"
	"crypto/rand"
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
	Name           string
}

type ContainerSpec struct {
	context string
	image   string
	runArgs []string
	setup   func(c *Container) error
	cleanup func(c *Container) error
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
	cleanup: removeHostKey,
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
	cleanup: removeHostKey,
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
	cleanup: removeHostKey,
}

func StartContainer(t *testing.T, spec ContainerSpec) *Container {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that requires a container in short mode")
	}
	finishPhase := MeasureTestPhase(t, "fixture: docker info")
	RequireLinuxDockerEngine(t)
	finishPhase()

	finishPhase = MeasureTestPhase(t, "fixture: build image "+spec.image)
	if err := buildImage(spec); err != nil {
		t.Fatalf("failed to build image: %v", err)
	}

	finishPhase()
	containerName := generateContainerName(t)
	t.Cleanup(func() {
		defer MeasureTestPhase(t, "cleanup: delete fixture container")()
		deleteContainer(containerName)
	})

	finishPhase = MeasureTestPhase(t, "fixture: start container and wait for port")
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

	finishPhase()
	c := &Container{
		SSHDestination: fmt.Sprintf("ssh://root@localhost:%s", port),
		Name:           containerName,
	}

	if spec.setup != nil {
		finishPhase = MeasureTestPhase(t, "fixture: host key and daemon readiness")
		if err := spec.setup(c); err != nil {
			t.Fatalf("container setup failed: %v", err)
		}
		finishPhase()
	}
	if spec.cleanup != nil {
		t.Cleanup(func() {
			defer MeasureTestPhase(t, "cleanup: fixture host key")()
			if err := spec.cleanup(c); err != nil {
				t.Errorf("container cleanup failed: %v", err)
			}
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

var knownHostsLockPath = filepath.Join(os.TempDir(), "topo-e2e-known_hosts.lock")

func acceptHostKey(c *Container) error {
	flock, err := AcquireFlock(knownHostsLockPath)
	if err != nil {
		return err
	}
	defer flock.Release()

	// #nosec G204 -- ignore as its a test helper
	cmd := exec.Command("ssh", c.SSHDestination, "-o", "StrictHostKeyChecking=accept-new", "true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w output: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func removeHostKey(c *Container) error {
	u, err := url.Parse(c.SSHDestination)
	if err != nil {
		return err
	}
	host := fmt.Sprintf("[%s]:%s", u.Hostname(), u.Port())
	flock, err := AcquireFlock(knownHostsLockPath)
	if err != nil {
		return err
	}
	defer flock.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		// #nosec G204 -- ignore as its a test helper
		output, err := exec.CommandContext(ctx, "ssh-keygen", "-R", host).CombinedOutput()
		if err == nil {
			return nil
		}
		removeErr := fmt.Errorf("remove host key: %w output: %s", err, strings.TrimSpace(string(output)))
		if runtime.GOOS != "windows" || !strings.Contains(string(output), "rename") || !strings.Contains(string(output), "Permission denied") {
			return removeErr
		}
		select {
		case <-ctx.Done():
			return removeErr
		case <-time.After(100 * time.Millisecond):
		}
	}
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
