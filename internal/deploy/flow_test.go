package deploy_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/podman"
	deploytestutil "github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	t.Run("Docker", func(t *testing.T) {
		testutil.RequireDocker(t)

		t.Run("deploys to localhost", func(t *testing.T) {
			scope, imageName := deploytestutil.DockerDeploymentFixture(t)
			t.Cleanup(func() { deploytestutil.CleanupDockerComposeProject(t, scope) })
			deploytestutil.RequireDockerImageDoesNotExist(t, docker.LocalHost, imageName)
			deployOptions := deploy.Options{Engine: deploy.EngineDocker, TargetHost: ssh.PlainLocalhost}

			err := deploy.Deploy(t.Context(), t.Output(), scope, deployOptions)

			require.NoError(t, err)
			deploytestutil.RequireDockerImageExists(t, docker.LocalHost, imageName)
			deploytestutil.AssertDockerContainersRunning(t, ssh.PlainLocalhost, scope)
		})

		t.Run("transfers images to a remote host via pipe", func(t *testing.T) {
			container := testutil.StartContainer(t, testutil.DinDContainer)
			remoteDockerHost := ssh.NewDestination(container.SSHDestination)
			scope, imageName := deploytestutil.DockerDeploymentFixture(t)
			deploytestutil.RequireDockerImageDoesNotExist(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
			deployOptions := deploy.Options{Engine: deploy.EngineDocker, TargetHost: remoteDockerHost}

			err := deploy.Deploy(t.Context(), t.Output(), scope, deployOptions)

			require.NoError(t, err)
			deploytestutil.RequireDockerImageExists(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
			deploytestutil.AssertDockerContainersRunning(t, remoteDockerHost, scope)
		})

		t.Run("transfers images to a remote host through a registry", func(t *testing.T) {
			registryContainerName := deploytestutil.TestContainerName(t) + "-registry"
			registryPort := "12738"
			container := testutil.StartContainer(t, testutil.DinDContainer)
			remoteDockerHost := ssh.NewDestination(container.SSHDestination)
			remoteCommandHost := docker.NewHostFromDestination(remoteDockerHost)
			scope, imageName := deploytestutil.DockerDeploymentFixture(t)
			deploytestutil.RequireDockerImageDoesNotExist(t, remoteCommandHost, imageName)
			deployOptions := deploy.Options{
				Engine:     deploy.EngineDocker,
				TargetHost: remoteDockerHost,
				Registry: &deploy.RegistryConfig{
					ContainerName:       registryContainerName,
					Port:                registryPort,
					SkipRemotePortCheck: true,
				},
			}

			err := deploy.Deploy(t.Context(), t.Output(), scope, deployOptions)

			require.NoError(t, err)
			deploytestutil.RequireDockerImageExists(t, remoteCommandHost, imageName)
			deploytestutil.AssertDockerContainersRunning(t, remoteDockerHost, scope)
		})
	})

	t.Run("Podman", func(t *testing.T) {
		t.Run("rejects runtime before accessing Podman", func(t *testing.T) {
			composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  firmware:
    image: alpine
    runtime: io.containerd.remoteproc.v1
`)

			scope := project.Scope{ComposeFile: composeFile}
			options := deploy.Options{Engine: deploy.EnginePodman}

			err := deploy.Deploy(t.Context(), &bytes.Buffer{}, scope, options)

			require.ErrorContains(t, err, `specifying "runtime:" in Compose files is unsupported for Podman deployments`)
		})

		t.Run("deploys to localhost", func(t *testing.T) {
			testutil.RequirePodman(t)
			scope, projectName := deploytestutil.PodmanDeploymentFixture(t)
			t.Cleanup(func() { deploytestutil.CleanupPodmanComposeProject(t, scope) })
			options := deploy.Options{Engine: deploy.EnginePodman, TargetHost: ssh.PlainLocalhost}

			err := deploy.Deploy(t.Context(), t.Output(), scope, options)

			require.NoError(t, err)
			deploytestutil.AssertPodmanContainersInState(t, projectName, podman.LocalSocket, "running")
		})

		t.Run("transfers images to a remote host via pipe", func(t *testing.T) {
			testutil.RequirePodman(t)
			podmanContainer := testutil.StartContainer(t, testutil.PodmanContainer)
			scope, projectName := deploytestutil.PodmanDeploymentFixture(t)
			targetDestination := ssh.NewDestination(podmanContainer.SSHDestination)
			options := deploy.Options{Engine: deploy.EnginePodman, TargetHost: targetDestination}

			err := deploy.Deploy(t.Context(), t.Output(), scope, options)

			require.NoError(t, err)
			tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetDestination)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tunnel.Close())
			})
			deploytestutil.AssertPodmanContainersInState(t, projectName, podman.NewSocket(tunnel.SocketURL()), "running")
		})

		t.Run("transfers images to a remote host through a registry", func(t *testing.T) {
			testutil.RequirePodman(t)
			registryContainerName := deploytestutil.TestContainerName(t) + "-registry"
			registryPort := "12739"
			podmanContainer := testutil.StartContainer(t, testutil.PodmanContainer)
			scope, projectName := deploytestutil.PodmanDeploymentFixture(t)
			targetDestination := ssh.NewDestination(podmanContainer.SSHDestination)
			options := deploy.Options{
				Engine:     deploy.EnginePodman,
				TargetHost: targetDestination,
				Registry: &deploy.RegistryConfig{
					ContainerName: registryContainerName,
					Port:          registryPort,
				},
			}

			err := deploy.Deploy(t.Context(), t.Output(), scope, options)

			require.NoError(t, err)
			tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetDestination)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tunnel.Close())
			})
			deploytestutil.AssertPodmanContainersInState(t, projectName, podman.NewSocket(tunnel.SocketURL()), "running")
		})
	})
}

func TestStop(t *testing.T) {
	t.Run("Docker", func(t *testing.T) {
		testutil.RequireDocker(t)

		container := testutil.StartContainer(t, testutil.DinDContainer)
		remoteDockerHost := ssh.NewDestination(container.SSHDestination)
		temporaryDirectory := t.TempDir()
		dockerFilePath := filepath.Join(temporaryDirectory, "Dockerfile")
		testutil.RequireWriteFile(t, dockerFilePath, `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
		composeFilePath := testutil.RequireWriteComposeFile(t, temporaryDirectory, fmt.Sprintf(`
name: %s
services:
  busybox:
    image: busybox
    command: ["tail", "-f", "/dev/null"]
    stop_grace_period: 1s
  a-service:
    build: .
    stop_grace_period: 1s
`, deploytestutil.TestProjectName(t)))
		scope := project.Scope{ComposeFile: composeFilePath}
		t.Cleanup(func() { deploytestutil.CleanupDockerComposeProject(t, scope) })
		deployOptions := deploy.Options{Engine: deploy.EngineDocker, TargetHost: remoteDockerHost}
		require.NoError(t, deploy.Deploy(t.Context(), io.Discard, scope, deployOptions))
		deploytestutil.AssertDockerContainersRunning(t, remoteDockerHost, scope)

		err := deploy.Stop(t.Context(), io.Discard, scope, remoteDockerHost, deploy.EngineDocker)

		require.NoError(t, err)
		deploytestutil.AssertDockerContainersStopped(t, remoteDockerHost, scope)
	})

	t.Run("Podman", func(t *testing.T) {
		testutil.RequirePodman(t)

		t.Run("stops services on localhost", func(t *testing.T) {
			scope, projectName := deploytestutil.PodmanDeploymentFixture(t)
			t.Cleanup(func() { deploytestutil.CleanupPodmanComposeProject(t, scope) })
			options := deploy.Options{Engine: deploy.EnginePodman, TargetHost: ssh.PlainLocalhost}
			require.NoError(t, deploy.Deploy(t.Context(), t.Output(), scope, options))

			err := deploy.Stop(t.Context(), t.Output(), scope, ssh.PlainLocalhost, deploy.EnginePodman)

			require.NoError(t, err)
			deploytestutil.AssertPodmanContainersInState(t, projectName, podman.LocalSocket, "exited")
		})

		t.Run("stops services on a remote target", func(t *testing.T) {
			podmanContainer := testutil.StartContainer(t, testutil.PodmanContainer)
			scope, projectName := deploytestutil.PodmanDeploymentFixture(t)
			target := ssh.NewDestination(podmanContainer.SSHDestination)
			options := deploy.Options{Engine: deploy.EnginePodman, TargetHost: target}
			require.NoError(t, deploy.Deploy(t.Context(), t.Output(), scope, options))

			err := deploy.Stop(t.Context(), t.Output(), scope, target, deploy.EnginePodman)

			require.NoError(t, err)
			tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), io.Discard, target)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tunnel.Close()) })
			deploytestutil.AssertPodmanContainersInState(t, projectName, podman.NewSocket(tunnel.SocketURL()), "exited")
		})
	})
}

func TestPrepareRegistry(t *testing.T) {
	config := deploy.RegistryConfig{ContainerName: "test-registry", Port: "12345"}

	t.Run("creates a missing registry with the requested configuration", func(t *testing.T) {
		executor := &registryExecutorFake{}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.NoError(t, err)
		assert.Equal(t, [][]string{
			{"inspect", "test-registry"},
			{
				"run", "--pull=missing", "-d", "--restart", "always",
				"-p", "127.0.0.1:12345:5000", "--name", "test-registry", "registry:2",
			},
		}, executor.runCalls)
		assert.Empty(t, executor.commandCalls)
	})

	t.Run("starts a stopped registry using its configured port", func(t *testing.T) {
		executor := &registryExecutorFake{
			exists: true,
			inspectOutput: `[{"State":{"Running":false},
				"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"12345"}]}},
				"NetworkSettings":{"Ports":{}}}]`,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.NoError(t, err)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}, {"start", "test-registry"}}, executor.runCalls)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.commandCalls)
	})

	t.Run("reuses a running registry using its engine-assigned port", func(t *testing.T) {
		executor := &registryExecutorFake{
			exists: true,
			inspectOutput: `[{"State":{"Running":true},
				"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"0"}]}},
				"NetworkSettings":{"Ports":{"5000/tcp":[{"HostPort":"12345"}]}}}]`,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.NoError(t, err)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}, {"start", "test-registry"}}, executor.runCalls)
	})

	t.Run("rejects a mismatched port without starting or creating a registry", func(t *testing.T) {
		executor := &registryExecutorFake{
			exists: true,
			inspectOutput: `[{"State":{"Running":true},
				"NetworkSettings":{"Ports":{"5000/tcp":[{"HostPort":"54321"}]}}}]`,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.EqualError(t, err,
			"registry port mismatch (running: 54321, requested: 12345)\n"+
				"you may need to remove the existing registry container test-registry")
		assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.runCalls)
	})

	t.Run("reports an inspection command failure without starting the registry", func(t *testing.T) {
		executor := &registryExecutorFake{exists: true, inspectExitCode: 1}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		require.ErrorContains(t, err, "failed to execute")
		var exitError *exec.ExitError
		assert.ErrorAs(t, err, &exitError)
		assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.runCalls)
	})

	t.Run("rejects invalid inspection data without starting the registry", func(t *testing.T) {
		tests := []struct {
			name          string
			inspectOutput string
			wantError     string
		}{
			{"malformed JSON", "not JSON", "failed to inspect existing registry test-registry: decode registry inspect output"},
			{"no containers", "[]", "expected one inspected container, got 0"},
			{"multiple containers", "[{},{}]", "expected one inspected container, got 2"},
			{"missing binding", "[{}]", "container port 5000 is not published"},
			{
				"empty bindings",
				`[{"HostConfig":{"PortBindings":{"5000/tcp":[]}}}]`,
				"container port 5000 is not published",
			},
			{
				"empty host port",
				`[{"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":""}]}}}]`,
				"container port 5000 is not published",
			},
			{
				"running registry without an assigned port",
				`[{"State":{"Running":true},
					"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"12345"}]}}}]`,
				"container port 5000 is not published",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				executor := &registryExecutorFake{exists: true, inspectOutput: test.inspectOutput}
				progress := term.NewProgress(io.Discard)

				err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

				require.ErrorContains(t, err, test.wantError)
				assert.Equal(t, [][]string{{"inspect", "test-registry"}}, executor.runCalls)
			})
		}
	})

	t.Run("propagates a registry start failure", func(t *testing.T) {
		startError := errors.New("cannot start registry")
		executor := &registryExecutorFake{
			exists:        true,
			inspectOutput: `[{"HostConfig":{"PortBindings":{"5000/tcp":[{"HostPort":"12345"}]}}}]`,
			runError:      startError,
		}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, nil)

		assert.ErrorIs(t, err, startError)
	})

	t.Run("adds a diagnostic for a known port conflict and preserves engine output", func(t *testing.T) {
		runError := errors.New("registry run failed")
		executor := &registryExecutorFake{runError: runError, runOutput: "Error: Address ALREADY in use\n"}
		var output bytes.Buffer
		progress := term.NewProgress(&output)
		knownErrors := []string{"already in use", "already allocated"}

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, knownErrors)

		require.ErrorIs(t, err, runError)
		assert.ErrorContains(t, err, "port is already in use, this could be an existing test-registry or another process")
		assert.Contains(t, output.String(), executor.runOutput)
	})

	t.Run("propagates an unrecognised creation failure without a port conflict diagnostic", func(t *testing.T) {
		runError := errors.New("registry run failed")
		executor := &registryExecutorFake{runError: runError, runOutput: "permission denied\n"}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, []string{"address already in use"})

		assert.Same(t, runError, err)
	})

	t.Run("does not diagnose a port conflict when the command succeeds", func(t *testing.T) {
		executor := &registryExecutorFake{runOutput: "address already in use\n"}
		progress := term.NewProgress(io.Discard)

		err := deploy.PrepareRegistry(t.Context(), progress, config, executor, []string{"address already in use"})

		assert.NoError(t, err)
	})
}

type registryExecutorFake struct {
	exists          bool
	inspectOutput   string
	inspectExitCode int
	runError        error
	runOutput       string
	runCalls        [][]string
	commandCalls    [][]string
}

func (fake *registryExecutorFake) Command(_ context.Context, args ...string) *exec.Cmd {
	fake.commandCalls = append(fake.commandCalls, args)
	return testutil.CmdWithOutput(fake.inspectOutput, fake.inspectExitCode)
}

func (fake *registryExecutorFake) RunCommand(_ context.Context, output io.Writer, args ...string) error {
	fake.runCalls = append(fake.runCalls, args)
	if args[0] == "inspect" {
		if !fake.exists {
			return errors.New("container does not exist")
		}
		return nil
	}
	if _, err := io.WriteString(output, fake.runOutput); err != nil {
		return err
	}
	return fake.runError
}

func TestTransferImagesViaRegistry(t *testing.T) {
	t.Run("transfers every image by digest and restores its original name on the target", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
name: transfer
services:
  built:
    build: .
  pulled:
    image: team/web:latest
`)
		scope := project.Scope{ComposeFile: composeFile}
		var calls [][]string
		source := &registryTransferFake{
			name: "source", calls: &calls,
			digests: map[string]string{
				"localhost:12345/team/web:latest": "localhost:12345/team/web@sha256:abc",
				"localhost:12345/transfer-built":  "localhost:12345/transfer-built@sha256:def",
			},
		}
		target := &registryTransferFake{name: "target", calls: &calls}
		progress := term.NewProgress(io.Discard)

		err := deploy.TransferImagesViaRegistry(t.Context(), progress, scope, "12345", source, target)

		require.NoError(t, err)
		want := [][]string{
			{"source", "tag", "team/web:latest", "localhost:12345/team/web:latest"},
			{"source", "push", "localhost:12345/team/web:latest"},
			{"target", "pull", "localhost:12345/team/web@sha256:abc"},
			{"target", "tag", "localhost:12345/team/web@sha256:abc", "team/web:latest"},

			{"source", "tag", "transfer-built", "localhost:12345/transfer-built"},
			{"source", "push", "localhost:12345/transfer-built"},
			{"target", "pull", "localhost:12345/transfer-built@sha256:def"},
			{"target", "tag", "localhost:12345/transfer-built@sha256:def", "transfer-built"},
		}
		assert.Equal(t, want, calls)
	})
}

type registryTransferFake struct {
	name    string
	calls   *[][]string
	digests map[string]string
}

func (fake *registryTransferFake) TagImage(_ context.Context, _ io.Writer, image, tag string) error {
	*fake.calls = append(*fake.calls, []string{fake.name, "tag", image, tag})
	return nil
}

func (fake *registryTransferFake) PushImage(_ context.Context, _ io.Writer, image string) (string, error) {
	*fake.calls = append(*fake.calls, []string{fake.name, "push", image})
	return fake.digests[image], nil
}

func (fake *registryTransferFake) PullImage(_ context.Context, _ io.Writer, digest string) error {
	*fake.calls = append(*fake.calls, []string{fake.name, "pull", digest})
	return nil
}
