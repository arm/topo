package health_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryExists(t *testing.T) {
	t.Run("wraps error as WarningError when severity is warning", func(t *testing.T) {
		check := health.BinaryExists{
			Severity: health.SeverityWarning,
		}
		dependency := health.Dependency{Binary: "nonexistent"}
		runner := &runner.Fake{}
		ctx := context.Background()

		_, err := check.Run(ctx, runner, dependency)

		wantErr := health.WarningError{Err: runner.BinaryExists(ctx, dependency.Binary)}
		assert.Equal(t, wantErr, err)
	})
}

func TestRemoveVersionChecks(t *testing.T) {
	t.Run("removes checks of type VersionMatches", func(t *testing.T) {
		dep := health.Dependency{
			Binary: "mixed",
			Label:  "Mixed",
			Checks: []health.Check{health.BinaryExists{}, health.VersionMatches{}},
		}

		got := health.RemoveVersionChecks([]health.Dependency{dep})

		assert.Len(t, got, 1)
		assert.Equal(t, got[0].Checks, []health.Check{health.BinaryExists{}})
	})
}

func TestVersionMatches(t *testing.T) {
	ctx := context.Background()
	dep := health.Dependency{}
	r := &runner.Fake{}

	t.Run("returns error when version is outdated", func(t *testing.T) {
		check := health.VersionMatches{
			FetchLatest: func(ctx context.Context) (string, error) {
				return "2.0.0", nil
			},
			CurrentVersion: "1.0.0",
		}

		_, err := check.Run(ctx, r, dep)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "1.0.0")
		assert.Contains(t, err.Error(), "2.0.0")
	})

	t.Run("returns nil when version matches latest", func(t *testing.T) {
		check := health.VersionMatches{
			FetchLatest: func(ctx context.Context) (string, error) {
				return "2.0.0", nil
			},
			CurrentVersion: "2.0.0",
		}

		fix, err := check.Run(ctx, r, dep)

		assert.NoError(t, err)
		assert.Empty(t, fix)
	})

	t.Run("degrades gracefully on fetch error", func(t *testing.T) {
		check := health.VersionMatches{
			FetchLatest: func(ctx context.Context) (string, error) {
				return "", fmt.Errorf("connection refused")
			},
		}

		fix, err := check.Run(ctx, r, dep)

		assert.NoError(t, err)
		assert.Empty(t, fix)
	})
}

func TestOpenSSHAvailable(t *testing.T) {
	ctx := context.Background()
	dependency := health.Dependency{Binary: "ssh", Label: "OpenSSH"}

	t.Run("accepts OpenSSH", func(t *testing.T) {
		check := health.OpenSSHAvailable{}
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"ssh -V": {Stderr: "OpenSSH_9.9p1, OpenSSL 3.4.0"},
		}}

		fix, err := check.Run(ctx, r, dependency)

		assert.NoError(t, err)
		assert.Nil(t, fix)
	})

	t.Run("rejects another SSH implementation", func(t *testing.T) {
		check := health.OpenSSHAvailable{}
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"ssh -V": {Stderr: "Dropbear v2025.88"},
		}}

		fix, err := check.Run(ctx, r, dependency)

		assert.EqualError(t, err, `"ssh" does not resolve to OpenSSH: Dropbear v2025.88`)
		assert.Equal(t, &health.Fix{
			Description: "Install OpenSSH and ensure its ssh executable is first on PATH",
		}, fix)
	})

	t.Run("returns an error when the version cannot be checked", func(t *testing.T) {
		check := health.OpenSSHAvailable{}
		versionErr := errors.New("version check failed")
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"ssh -V": {Err: versionErr},
		}}

		fix, err := check.Run(ctx, r, dependency)

		assert.ErrorIs(t, err, versionErr)
		assert.Nil(t, fix)
	})
}
func TestDockerComposeCompatible(t *testing.T) {
	ctx := context.Background()
	dep := health.Dependency{}

	t.Run("accepts Docker Compose at the minimum version", func(t *testing.T) {
		check := health.DockerComposeCompatible{MinVersion: "2.0.0"}
		runner := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"docker compose version": {Output: "Docker Compose version 2.0.0"},
			},
		}

		fix, err := check.Run(ctx, runner, dep)

		assert.NoError(t, err)
		assert.Nil(t, fix)
	})

	t.Run("accepts Docker Compose newer than the minimum version", func(t *testing.T) {
		check := health.DockerComposeCompatible{MinVersion: "2.0.0"}
		runner := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"docker compose version": {Output: "Docker Compose version 5.2.0"},
			},
		}

		fix, err := check.Run(ctx, runner, dep)

		assert.NoError(t, err)
		assert.Nil(t, fix)
	})

	t.Run("returns a plugin installation fix when the version command fails", func(t *testing.T) {
		check := health.DockerComposeCompatible{MinVersion: "2.0.0"}
		runner := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"docker compose version": {
					Err: fmt.Errorf("it blew up"),
				},
			},
		}

		fix, err := check.Run(ctx, runner, dep)

		assert.EqualError(t, err, "it blew up")
		assert.Equal(t, fix.Description, "Ensure Docker Compose is installed as a plugin for Docker")
	})

	t.Run("joins multi-line executions error into a one line", func(t *testing.T) {
		stderr := `docker: unknown command: docker compose

Run 'docker --help' for more information`
		check := health.DockerComposeCompatible{MinVersion: "2.0.0"}
		runner := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"docker compose version": {
					Err: errors.New(stderr),
				},
			},
		}

		_, err := check.Run(ctx, runner, dep)

		assert.EqualError(t, err, strings.ReplaceAll(stderr, "\n", " "))
	})

	t.Run("returns an upgrade fix when Docker Compose is too old", func(t *testing.T) {
		check := health.DockerComposeCompatible{MinVersion: "2.0.0"}
		runner := &runner.Fake{
			Commands: map[string]runner.FakeResult{
				"docker compose version": {Output: "Docker Compose version 1.9.0"},
			},
		}

		fix, err := check.Run(ctx, runner, dep)

		assert.EqualError(t, err, "installed docker compose version \"Docker Compose version 1.9.0\" is older than required version \"2.0.0\"")
		assert.Contains(t, fix.Description, "Upgrade Docker Compose")
	})
}
