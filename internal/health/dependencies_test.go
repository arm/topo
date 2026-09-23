package health_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestNewConnectivityDependency(t *testing.T) {
	t.Run("uses injected known hosts entry for changed host keys", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewConnectivityDependency(target, health.ConnectivityOperations{
			Authenticate:    func(context.Context, ssh.Destination) error { return ssh.ErrHostKeyChanged },
			KnownHostsEntry: func(ssh.Destination) (string, error) { return "[example.com]:2222", nil },
		})

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "host key has changed",
			Fix: &health.Fix{
				Description: "Remove the old SSH host key from known_hosts, then retry",
				Command:     "ssh-keygen -R '[example.com]:2222'",
			},
		}}
		assert.Equal(t, want, got)
	})

	t.Run("omits the removal command when the known hosts entry cannot be resolved", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewConnectivityDependency(target, health.ConnectivityOperations{
			Authenticate:    func(context.Context, ssh.Destination) error { return ssh.ErrHostKeyChanged },
			KnownHostsEntry: func(ssh.Destination) (string, error) { return "", errors.New("cannot load SSH config") },
		})

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "host key has changed",
			Fix: &health.Fix{
				Description: "Remove the old SSH host key from known_hosts, then retry",
			},
		}}
		assert.Equal(t, want, got)
	})

	t.Run("returns setup keys advice for authentication failures", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewConnectivityDependency(target, health.ConnectivityOperations{
			Authenticate: func(context.Context, ssh.Destination) error { return ssh.ErrAuthFailed },
		})

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "authentication failed",
			Fix: &health.Fix{
				Description: "Configure SSH keys on remote target",
				Command:     "topo setup-keys --target ssh://user@example.com",
			},
		}}
		assert.Equal(t, want, got)
	})

	t.Run("returns setup keys advice for too many authentication failures", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewConnectivityDependency(target, health.ConnectivityOperations{
			Authenticate: func(context.Context, ssh.Destination) error { return ssh.ErrTooManyAuthFails },
		})

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "too many authentication failures",
			Fix: &health.Fix{
				Description: "Configure SSH keys on remote target",
				Command:     "topo setup-keys --target ssh://user@example.com",
			},
		}}
		assert.Equal(t, want, got)
	})

	t.Run("returns host key trust advice for unknown host keys", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewConnectivityDependency(target, health.ConnectivityOperations{
			Authenticate: func(context.Context, ssh.Destination) error { return ssh.ErrHostKeyUnknown },
		})

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "host key is unknown",
			Fix: &health.Fix{
				Description: "Trust the target's SSH host key",
				Command:     "topo health --target ssh://user@example.com --accept-new-host-keys",
			},
		}}
		assert.Equal(t, want, got)
	})
}

func TestNewDependencyOnRemoteDockerDaemon(t *testing.T) {
	t.Run("reports a reachable daemon", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewDependencyOnRemoteDockerDaemon(target, func(context.Context, ssh.Destination) error {
			return nil
		})

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "reachable"}, got)
	})

	t.Run("reports a failed daemon probe", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewDependencyOnRemoteDockerDaemon(target, func(context.Context, ssh.Destination) error {
			return errors.New("Boom!")
		})

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "Boom!",
			Fix:      &health.Fix{Description: "Ensure docker is installed and running on the target. See https://github.com/arm/topo#install-a-container-engine"},
		}}
		assert.Equal(t, want, got)
	})
}

func TestNewDependencyOnSSHCheck(t *testing.T) {
	buildRunner := func(sshVResult runner.FakeResult) runner.Runner {
		return &runner.Fake{
			Binaries: []string{"ssh"},
			Commands: map[string]runner.FakeResult{"ssh -V": sshVResult},
		}
	}

	t.Run("accepts OpenSSH", func(t *testing.T) {
		dependency := health.NewDependencyOnSSH(buildRunner(runner.FakeResult{Stderr: "OpenSSH_9.9p1, OpenSSL 3.4.0"}))

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "ssh"}, got)
	})

	t.Run("rejects another SSH implementation", func(t *testing.T) {
		dependency := health.NewDependencyOnSSH(buildRunner(runner.FakeResult{Stderr: "Dropbear v2025.88"}))

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Message: `"ssh" does not resolve to OpenSSH: Dropbear v2025.88`,
			Fix:     &health.Fix{Description: "Install OpenSSH and ensure its ssh executable is first on PATH"},
		}}
		assert.Equal(t, want, got)
	})

	t.Run("fails when the version cannot be checked", func(t *testing.T) {
		versionErr := errors.New("version check failed")
		dependency := health.NewDependencyOnSSH(buildRunner(runner.FakeResult{Err: versionErr}))

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{Message: versionErr.Error()}}, got)
	})
}

func TestNewDependencyOnDockerComposeCheck(t *testing.T) {
	buildRunner := func(version string) runner.Runner {
		return &runner.Fake{Commands: map[string]runner.FakeResult{
			"docker compose version --format json": {Output: `{"version": "` + version + `"}`},
		}}
	}

	t.Run("accepts Docker Compose at the minimum version", func(t *testing.T) {
		dependency := health.NewDependencyOnDockerCompose(buildRunner("2.21.0"))

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "docker compose"}, got)
	})

	t.Run("accepts Docker Compose newer than the minimum version", func(t *testing.T) {
		dependency := health.NewDependencyOnDockerCompose(buildRunner("5.2.0"))

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "docker compose"}, got)
	})

	t.Run("returns an upgrade fix when Docker Compose is too old", func(t *testing.T) {
		dependency := health.NewDependencyOnDockerCompose(buildRunner("v1.9.0"))

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Message: "installed docker compose version v1.9.0 is older than required version 2.21.0",
			Fix:     &health.Fix{Description: "Upgrade Docker Compose to version 2.21.0 or later. See https://github.com/arm/topo#install-a-container-engine"},
		}}
		assert.Equal(t, want, got)
	})
}

func TestRemoteprocDependency(t *testing.T) {
	buildRunnerWithRemoteProcs := func(names []string) runner.Runner {
		return &runner.Fake{Commands: map[string]runner.FakeResult{
			"cat /sys/class/remoteproc/*/name": {Output: strings.Join(names, "\n")},
		}}
	}

	t.Run("fails when no remoteproc devices are found", func(t *testing.T) {
		r := buildRunnerWithRemoteProcs(nil)
		dependency := health.NewDependencyOnRemoteproc(r)

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{
			Failure: &health.DependencyCheckFailure{
				Severity: health.SeverityInfo,
				Message:  "no remoteproc devices found",
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("fails when remoteproc probe fails", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"cat /sys/class/remoteproc/*/name": {Err: runner.ErrTimeout},
		}}
		dependency := health.NewDependencyOnRemoteproc(r)

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{
			Failure: &health.DependencyCheckFailure{
				Severity: health.SeverityError,
				Message:  "timed out",
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("reports remoteproc device names", func(t *testing.T) {
		r := buildRunnerWithRemoteProcs([]string{"m4_0", "m4_1"})
		dependency := health.NewDependencyOnRemoteproc(r)

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "m4_0, m4_1"}, got)
	})
}

func TestRemoteprocRuntimeDependency(t *testing.T) {
	t.Run("includes an install fix with the target", func(t *testing.T) {
		target := ssh.NewDestination("user@my-target")
		dependency := health.NewDependencyOnRemoteprocRuntime(target, &runner.Fake{})

		result := dependency.Check(context.Background())

		assert.Equal(t, &health.DependencyCheckFailure{
			Severity: health.SeverityWarning,
			Message:  `"remoteproc-runtime" not found in $PATH`,
			Fix: &health.Fix{
				Description: "Install the Remoteproc Runtime",
				Command:     "topo install remoteproc-runtime --target ssh://user@my-target",
			},
		}, result.Failure)
	})
}

func TestRemoteprocRuntimeShimDependency(t *testing.T) {
	t.Run("includes an install fix with the target", func(t *testing.T) {
		target := ssh.NewDestination("user@my-target")
		dependency := health.NewDependencyOnRemoteprocRuntimeShim(target, &runner.Fake{})

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityWarning,
			Message:  `"containerd-shim-remoteproc-v1" not found in $PATH`,
			Fix: &health.Fix{
				Description: "Install the Remoteproc Runtime",
				Command:     "topo install remoteproc-runtime --target ssh://user@my-target",
			},
		}}, got)
	})
}

const podmanInstallURL = "https://github.com/arm/topo#install-a-container-engine"

func TestNewDependencyOnPodmanCLI(t *testing.T) {
	t.Run("reports an available Podman binary", func(t *testing.T) {
		dependency := health.NewDependencyOnPodmanCLI(&runner.Fake{Binaries: []string{"podman"}})

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "podman"}, got)
	})

	t.Run("reports a missing Podman binary", func(t *testing.T) {
		dependency := health.NewDependencyOnPodmanCLI(&runner.Fake{})

		got := dependency.Check(t.Context())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  `"podman" not found in $PATH`,
			Fix:      &health.Fix{Description: "Install Podman. See " + podmanInstallURL},
		}}
		assert.Equal(t, want, got)
	})
}

func TestNewDependencyOnPodmanConnection(t *testing.T) {
	t.Run("reports a reachable Podman connection", func(t *testing.T) {
		dependency := health.NewDependencyOnPodmanConnection(&runner.Fake{Commands: map[string]runner.FakeResult{"podman info": {}}})

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "reachable"}, got)
	})

	t.Run("reports an unavailable Podman connection", func(t *testing.T) {
		dependency := health.NewDependencyOnPodmanConnection(&runner.Fake{Commands: map[string]runner.FakeResult{"podman info": {Err: errors.New("permission denied")}}})

		got := dependency.Check(t.Context())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "permission denied",
			Fix:      &health.Fix{Description: "Start a Podman machine or configure an active Podman system connection, then ensure the current user can run Podman commands. See " + podmanInstallURL},
		}}
		assert.Equal(t, want, got)
	})
}

func TestNewDependencyOnDockerComposeForPodman(t *testing.T) {
	t.Run("reports an available Compose provider", func(t *testing.T) {
		dependency := health.NewDependencyOnDockerComposeForPodman(func(context.Context) error { return nil })

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "docker-compose"}, got)
	})

	t.Run("reports an unavailable Compose provider", func(t *testing.T) {
		dependency := health.NewDependencyOnDockerComposeForPodman(func(context.Context) error { return errors.New("version failed") })

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "version failed",
			Fix:      &health.Fix{Description: "Ensure docker-compose is on the $PATH. See " + podmanInstallURL},
		}}, got)
	})
}

func TestNewDependencyOnRemotePodmanAPI(t *testing.T) {
	t.Run("reports a successful probe", func(t *testing.T) {
		dependency := health.NewDependencyOnRemotePodmanAPI(func(context.Context) probe.RemotePodmanProbeResult {
			return probe.RemotePodmanProbeResult{SocketPath: "/run/podman.sock"}
		})

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "/run/podman.sock"}, got)
	})

	t.Run("reports a timeout", func(t *testing.T) {
		result := probe.RemotePodmanProbeResult{Err: fmt.Errorf("%w: %w", probe.ErrRemotePodmanSocketResolutionFailed, context.DeadlineExceeded)}
		dependency := health.NewDependencyOnRemotePodmanAPI(func(context.Context) probe.RemotePodmanProbeResult { return result })
		ctx, cancel := context.WithTimeout(t.Context(), 0)
		defer cancel()

		got := dependency.Check(ctx)

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  "health check timed out",
			Fix:      &health.Fix{Description: "Retry the health check with a longer timeout."},
		}}, got)
	})

	t.Run("reports a socket resolution failure", func(t *testing.T) {
		result := probe.RemotePodmanProbeResult{Err: fmt.Errorf("%w: %w", probe.ErrRemotePodmanSocketResolutionFailed, errors.New("no remote socket"))}
		dependency := health.NewDependencyOnRemotePodmanAPI(func(context.Context) probe.RemotePodmanProbeResult { return result })

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  result.Err.Error(),
			Fix:      &health.Fix{Description: "Start the Podman API socket and ensure the SSH user can access it. See " + podmanInstallURL},
		}}, got)
	})

	t.Run("reports a forwarding failure", func(t *testing.T) {
		result := probe.RemotePodmanProbeResult{
			SocketPath: "/run/podman.sock",
			Err:        fmt.Errorf("%w: %w", probe.ErrRemotePodmanForwardingFailed, errors.New("forwarding denied")),
		}
		dependency := health.NewDependencyOnRemotePodmanAPI(func(context.Context) probe.RemotePodmanProbeResult { return result })

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  result.Err.Error(),
			Fix:      &health.Fix{Description: "Ensure the target SSH server permits local TCP forwarding to the target Podman API socket at /run/podman.sock."},
		}}, got)
	})

	t.Run("reports an API request failure", func(t *testing.T) {
		result := probe.RemotePodmanProbeResult{
			SocketPath: "/run/podman.sock",
			Err:        fmt.Errorf("%w: %w", probe.ErrRemotePodmanAPIRequestFailed, errors.New("request failed")),
		}
		dependency := health.NewDependencyOnRemotePodmanAPI(func(context.Context) probe.RemotePodmanProbeResult { return result })

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  result.Err.Error(),
			Fix:      &health.Fix{Description: "Ensure the Podman API socket at /run/podman.sock is functional and accessible to the SSH user."},
		}}, got)
	})

	t.Run("reports a cleanup failure without a fix", func(t *testing.T) {
		result := probe.RemotePodmanProbeResult{
			SocketPath: "/run/podman.sock",
			Err:        fmt.Errorf("%w: %w", probe.ErrRemotePodmanSocketTunnelCloseFailed, errors.New("close failed")),
		}
		dependency := health.NewDependencyOnRemotePodmanAPI(func(context.Context) probe.RemotePodmanProbeResult { return result })

		got := dependency.Check(t.Context())

		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  result.Err.Error(),
		}}, got)
	})
}
