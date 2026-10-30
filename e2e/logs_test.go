package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsValidation(t *testing.T) {
	topo := buildBinary(t)
	t.Setenv("TOPO_TARGET", "")
	t.Setenv("TOPO_ENGINE", "")
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"requires a target", nil, "target not specified"},
		{"rejects invalid engine", []string{"--engine", "invalid"}, `invalid engine "invalid"`},
		{"requires a compose file", []string{"--target", "localhost"}, "compose.yaml"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command(topo, append([]string{"logs"}, test.args...)...)
			cmd.Dir = t.TempDir()

			output, err := cmd.CombinedOutput()

			require.Error(t, err)
			assert.Contains(t, string(output), test.want)
		})
	}
}

func TestLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container logs tests in short mode")
	}
	topo := buildBinary(t)

	t.Run("local Docker", func(t *testing.T) {
		testutil.RequireLinuxDockerEngine(t)
		runner := docker.Engine{Host: docker.LocalHost}

		testLogs(t, topo, deploy.EngineDocker, ssh.PlainLocalhost, runner)
	})

	t.Run("remote Docker", func(t *testing.T) {
		container := testutil.StartContainer(t, testutil.DinDContainer)
		target := ssh.NewDestination(container.SSHDestination)
		runner := docker.Engine{Host: docker.NewHostFromDestination(target)}

		testLogs(t, topo, deploy.EngineDocker, target, runner)
	})

	t.Run("local Podman", func(t *testing.T) {
		testutil.RequirePodman(t)
		runner := podman.Engine{Socket: podman.LocalSocket}

		testLogs(t, topo, deploy.EnginePodman, ssh.PlainLocalhost, runner)
	})

	t.Run("remote Podman", func(t *testing.T) {
		testutil.RequirePodman(t)
		container := testutil.StartContainer(t, testutil.PodmanContainer)
		target := ssh.NewDestination(container.SSHDestination)
		tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), target)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tunnel.Close()) })
		runner := podman.Engine{Socket: podman.NewSocket(tunnel.SocketURL())}

		testLogs(t, topo, deploy.EnginePodman, target, runner)
	})
}

func testLogs(t *testing.T, topo string, engine deploy.EngineKind, target ssh.Destination, runner deploy.ComposeCommandRunner) {
	t.Helper()
	scope := logsProject(t)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		require.NoError(t, runner.RunComposeCommand(ctx, t.Output(), scope, "down", "--volumes"))
	})
	require.NoError(t, deploy.Deploy(t.Context(), t.Output(), scope, deploy.Options{Engine: engine, TargetHost: target}))
	args := []string{
		"logs", "-f", scope.ComposeFile,
		"--target", target.String(),
		"--engine", string(engine),
	}

	t.Run("shows logs from all services", func(t *testing.T) {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			output, err := runLogsCommand(t, topo, args)

			assert.NoError(c, err, output)
			assert.Contains(c, output, "stdout-first")
			assert.Contains(c, output, "stderr-last")
			assert.Contains(c, output, "other-service")
		}, 30*time.Second, 200*time.Millisecond)
	})

	t.Run("reports an unknown service", func(t *testing.T) {
		output, err := runLogsCommand(t, topo, args, "unknown-service")

		require.Error(t, err)
		assert.Contains(t, output, "no such service")
	})

	t.Run("streams output and exits cleanly on interrupt", func(t *testing.T) {
		assertLogsFollow(t, topo, args)
	})

	t.Run("shows only the selected service's logs after stopping", func(t *testing.T) {
		require.NoError(t, runner.RunComposeCommand(t.Context(), t.Output(), scope, "stop"))

		output, err := runLogsCommand(t, topo, args, "logger")

		require.NoError(t, err, output)
		assert.Contains(t, output, "stdout-first")
		assert.Contains(t, output, "stderr-last")
		assert.NotContains(t, output, "other-service")
	})
}

func logsProject(t *testing.T) project.Scope {
	t.Helper()
	composeFile := filepath.Join(t.TempDir(), "logs.yaml")
	contents := fmt.Sprintf(`name: %s
services:
  logger:
    image: docker.io/library/busybox:latest
    command: ["sh", "-c", "echo stdout-first; echo stderr-last >&2; exec sleep 300"]
    stop_grace_period: 1s
  other:
    image: docker.io/library/busybox:latest
    command: ["sh", "-c", "echo other-service; exec sleep 300"]
    stop_grace_period: 1s
`, "topo-"+testutil.SanitiseTestName(t))
	contents, err := testutil.FixPodmanInDockerQuirk(contents)
	require.NoError(t, err)
	testutil.RequireWriteFile(t, composeFile, contents)
	return project.Scope{ComposeFile: composeFile}
}

func runLogsCommand(t *testing.T, topo string, args []string, extra ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, topo, slices.Concat(args, extra)...)
	cmd.WaitDelay = 3 * time.Second
	if runtime.GOOS != "windows" {
		cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	}
	cmd.Dir = t.TempDir()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func assertLogsFollow(t *testing.T, topo string, args []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("sending os.Interrupt to a child process is unsupported on Windows")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, topo, slices.Concat(args, []string{"--follow", "logger"})...)
	cmd.WaitDelay = 3 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stopClosingOutput := context.AfterFunc(ctx, func() { _ = output.Close() })
	defer stopClosingOutput()
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "stdout-first") {
			break
		}
	}
	signalErr := cmd.Process.Signal(os.Interrupt)
	_, readErr := io.Copy(io.Discard, output)
	waitErr := cmd.Wait()

	assert.Contains(t, scanner.Text(), "stdout-first", "log output must arrive while the command is still running")
	assert.NoError(t, scanner.Err())
	assert.NoError(t, signalErr)
	assert.NoError(t, readErr)
	assert.NoError(t, waitErr, stderr.String())
}
