package health_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestNewMissingTargetDependency(t *testing.T) {
	t.Run("reports the configured severity and fix", func(t *testing.T) {
		dependency := health.NewMissingTargetDependency("target not specified", health.SeverityInfo, "Specify a target")

		got := dependency.Check(context.Background())

		want := health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityInfo,
			Message:  "target not specified",
			Fix:      &health.Fix{Description: "Specify a target"},
		}}
		assert.Equal(t, want, got)
	})
}

func TestNewConnectivityDependency(t *testing.T) {
	t.Run("uses injected known hosts entry for changed host keys", func(t *testing.T) {
		target := ssh.NewDestination("user@example.com")
		dependency := health.NewConnectivityDependency(target, health.ConnectivityOperations{
			Authenticate:    func(context.Context) error { return ssh.ErrHostKeyChanged },
			KnownHostsEntry: func() (string, error) { return "[example.com]:2222", nil },
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
			Authenticate:    func(context.Context) error { return ssh.ErrHostKeyChanged },
			KnownHostsEntry: func() (string, error) { return "", errors.New("cannot load SSH config") },
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
			Authenticate: func(context.Context) error { return ssh.ErrAuthFailed },
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
			Authenticate: func(context.Context) error { return ssh.ErrTooManyAuthFails },
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
			Authenticate: func(context.Context) error { return ssh.ErrHostKeyUnknown },
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

func TestNewDependencyOnRemoteDocker(t *testing.T) {
	t.Run("does not probe the target when Docker is missing on the host", func(t *testing.T) {
		probeCalled := false
		dependency := health.NewDependencyOnRemoteDocker(&runner.Fake{}, func(context.Context) error {
			probeCalled = true
			return nil
		})

		got := dependency.Check(context.Background())

		assert.False(t, probeCalled)
		assert.Equal(t, health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
			Severity: health.SeverityError,
			Message:  `cannot probe from host: "docker" not found in $PATH`,
			Fix:      &health.Fix{Description: "Install a supported container engine on the host. See https://github.com/arm/topo#install-a-container-engine"},
		}}, got)
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
		dependency := health.NewDependencyOnRemoteprocRuntime(ssh.NewDestination("user@my-target"), &runner.Fake{})

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
