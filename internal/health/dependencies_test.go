package health_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/version"
	"github.com/stretchr/testify/assert"
)

func TestNewDependencyOnTopoCheck(t *testing.T) {
	t.Run("passes for development builds", func(t *testing.T) {
		originalVersion := version.Version
		version.Version = version.Dev
		t.Cleanup(func() { version.Version = originalVersion })

		dependency := health.NewDependencyOnTopo(false)

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "topo"}, dependency.Check(context.Background()))
	})
}

func TestNewDependencyOnSSHCheck(t *testing.T) {
	buildRunner := func(result runner.FakeResult) runner.Runner {
		return &runner.Fake{
			Binaries: []string{"ssh"},
			Commands: map[string]runner.FakeResult{"ssh -V": result},
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
			"docker-compose":                       {},
			"docker compose version --format json": {Output: `{"version": "` + version + `"}`},
		}}
	}

	t.Run("accepts Docker Compose at the minimum version", func(t *testing.T) {
		dependency := health.NewDependencyOnDockerCompose(buildRunner("2.21.0"))

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "docker-compose"}, got)
	})

	t.Run("accepts Docker Compose newer than the minimum version", func(t *testing.T) {
		dependency := health.NewDependencyOnDockerCompose(buildRunner("5.2.0"))

		got := dependency.Check(context.Background())

		assert.Equal(t, health.DependencyCheckResult{SuccessValue: "docker-compose"}, got)
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
	t.Run("Check", func(t *testing.T) {
		buildRunnerWithRemoteProcs := func(names []string) runner.Runner {
			return &runner.Fake{Commands: map[string]runner.FakeResult{
				"cat /sys/class/remoteproc/*/name": {Output: strings.Join(names, "\n")},
			}}
		}

		t.Run("fails when no remoteproc devices are found", func(t *testing.T) {
			r := buildRunnerWithRemoteProcs(nil)
			d := health.NewDependencyOnRemoteproc(r)

			got := d.Check(context.Background())

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
			d := health.NewDependencyOnRemoteproc(r)

			got := d.Check(context.Background())

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
			d := health.NewDependencyOnRemoteproc(r)

			got := d.Check(context.Background())

			assert.Equal(t, health.DependencyCheckResult{SuccessValue: "m4_0, m4_1"}, got)
		})
	})
}

func TestRemoteprocRuntimeDependency(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		t.Run("includes an install fix with the target", func(t *testing.T) {
			dep := health.NewDependencyOnRemoteprocRuntime(ssh.NewDestination("user@my-target"), &runner.Fake{})

			result := dep.Check(context.Background())

			assert.Equal(t, &health.DependencyCheckFailure{
				Severity: health.SeverityWarning,
				Message:  `"remoteproc-runtime" not found in $PATH`,
				Fix: &health.Fix{
					Description: "Install the Remoteproc Runtime",
					Command:     "topo install remoteproc-runtime --target ssh://user@my-target",
				},
			}, result.Failure)
		})
	})
}

func passingCheck(_ context.Context) health.DependencyCheckResult {
	return health.DependencyCheckResult{SuccessValue: "passed"}
}

func failingCheck(_ context.Context) health.DependencyCheckResult {
	return health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
		Severity: health.SeverityError,
		Message:  "very broken",
		Fix:      &health.Fix{Description: "fix me please", Command: "echo fixed"},
	}}
}
