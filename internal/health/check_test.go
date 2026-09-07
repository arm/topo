package health_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
)

func TestBinaryExists(t *testing.T) {
	t.Run("returns a warning result when severity is warning", func(t *testing.T) {
		check := health.BinaryExists{Severity: health.SeverityWarning}
		dependency := health.Dependency{Binary: "nonexistent"}
		runner := &runner.Fake{}

		got := check.Run(context.Background(), runner, dependency)

		want := &health.CheckFailure{
			Severity: health.SeverityWarning,
			Message:  runner.BinaryExists(context.Background(), dependency.Binary).Error(),
		}
		assert.Equal(t, want, got)
	})
}

func TestRemoveVersionChecks(t *testing.T) {
	t.Run("removes checks of type VersionMatches", func(t *testing.T) {
		dep := health.Dependency{Binary: "mixed", Label: "Mixed", Checks: []health.Check{health.BinaryExists{}, health.VersionMatches{}}}

		got := health.RemoveVersionChecks([]health.Dependency{dep})

		want := []health.Check{health.BinaryExists{}}

		assert.Len(t, got, 1)
		assert.Equal(t, want, got[0].Checks)
	})
}

func TestVersionMatches(t *testing.T) {
	ctx := context.Background()
	dep := health.Dependency{}
	r := &runner.Fake{}

	t.Run("returns an info result when version is outdated", func(t *testing.T) {
		check := health.VersionMatches{FetchLatest: func(context.Context) (string, error) { return "2.0.0", nil }, CurrentVersion: "1.0.0"}

		got := check.Run(ctx, r, dep)
		want := &health.CheckFailure{
			Severity: health.SeverityInfo,
			Message:  "out of date - current: 1.0.0, latest version: 2.0.0",
			Fix:      &health.Fix{},
		}

		assert.Equal(t, want, got)
	})

	t.Run("passes when version matches latest", func(t *testing.T) {
		check := health.VersionMatches{FetchLatest: func(context.Context) (string, error) { return "2.0.0", nil }, CurrentVersion: "2.0.0"}

		got := check.Run(ctx, r, dep)
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("passes when fetching the latest version fails", func(t *testing.T) {
		check := health.VersionMatches{FetchLatest: func(context.Context) (string, error) { return "", fmt.Errorf("connection refused") }}

		got := check.Run(ctx, r, dep)
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})
}

func TestOpenSSHAvailable(t *testing.T) {
	ctx := context.Background()
	dependency := health.Dependency{Binary: "ssh", Label: "OpenSSH"}

	t.Run("accepts OpenSSH", func(t *testing.T) {
		check := health.OpenSSHAvailable{}
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Stderr: "OpenSSH_9.9p1, OpenSSL 3.4.0"}}}

		got := check.Run(ctx, r, dependency)
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("rejects another SSH implementation", func(t *testing.T) {
		check := health.OpenSSHAvailable{}
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Stderr: "Dropbear v2025.88"}}}

		got := check.Run(ctx, r, dependency)
		want := &health.CheckFailure{
			Message: `"ssh" does not resolve to OpenSSH: Dropbear v2025.88`,
			Fix:     &health.Fix{Description: "Install OpenSSH and ensure its ssh executable is first on PATH"},
		}

		assert.Equal(t, want, got)
	})

	t.Run("fails when the version cannot be checked", func(t *testing.T) {
		check := health.OpenSSHAvailable{}
		versionErr := errors.New("version check failed")
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Err: versionErr}}}

		got := check.Run(ctx, r, dependency)
		want := &health.CheckFailure{Message: versionErr.Error()}

		assert.Equal(t, want, got)
	})
}

func TestDockerComposeCompatible(t *testing.T) {
	ctx := context.Background()
	dep := health.Dependency{}

	t.Run("accepts Docker Compose at the minimum version", func(t *testing.T) {
		check := health.DockerComposeMinVersion{MinVersion: "2.0.0"}
		runner := &runner.Fake{Commands: map[string]runner.FakeResult{"docker compose version --format json": {Output: `{"version": "2.0.0"}`}}}

		got := check.Run(ctx, runner, dep)
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("accepts Docker Compose newer than the minimum version", func(t *testing.T) {
		check := health.DockerComposeMinVersion{MinVersion: "2.0.0"}
		runner := &runner.Fake{Commands: map[string]runner.FakeResult{"docker compose version --format json": {Output: `{"version": "5.2.0"}`}}}

		got := check.Run(ctx, runner, dep)
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("returns an upgrade fix when Docker Compose is too old", func(t *testing.T) {
		check := health.DockerComposeMinVersion{MinVersion: "2.0.0"}
		runner := &runner.Fake{Commands: map[string]runner.FakeResult{"docker compose version --format json": {Output: `{"version": "v1.9.0"}`}}}

		got := check.Run(ctx, runner, dep)
		want := &health.CheckFailure{
			Message: "installed docker compose version v1.9.0 is older than required version 2.0.0",
			Fix:     &health.Fix{Description: "Upgrade Docker Compose to version 2.0.0 or later. See https://github.com/arm/topo#install-a-container-engine"},
		}

		assert.Equal(t, want, got)
	})
}
