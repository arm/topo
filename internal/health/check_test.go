package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/version"
	"github.com/stretchr/testify/assert"
)

func TestCheckTopoIsUpToDate(t *testing.T) {
	t.Run("passes for development builds", func(t *testing.T) {
		originalVersion := version.Version
		version.Version = version.Dev
		t.Cleanup(func() { version.Version = originalVersion })

		got := health.CheckTopoIsUpToDate(context.Background())

		assert.Nil(t, got)
	})
}

func TestCheckOpenSSHAvailable(t *testing.T) {
	ctx := context.Background()
	t.Run("accepts OpenSSH", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Stderr: "OpenSSH_9.9p1, OpenSSL 3.4.0"}}}

		got := health.CheckOpenSSHAvailable(ctx, r, "ssh")
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("rejects another SSH implementation", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Stderr: "Dropbear v2025.88"}}}

		got := health.CheckOpenSSHAvailable(ctx, r, "ssh")
		want := &health.CheckFailure{
			Message: `"ssh" does not resolve to OpenSSH: Dropbear v2025.88`,
			Fix:     &health.Fix{Description: "Install OpenSSH and ensure its ssh executable is first on PATH"},
		}

		assert.Equal(t, want, got)
	})

	t.Run("fails when the version cannot be checked", func(t *testing.T) {
		versionErr := errors.New("version check failed")
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Err: versionErr}}}

		got := health.CheckOpenSSHAvailable(ctx, r, "ssh")
		want := &health.CheckFailure{Message: versionErr.Error()}

		assert.Equal(t, want, got)
	})
}

func TestCheckDockerComposeMinVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("accepts Docker Compose at the minimum version", func(t *testing.T) {
		runner := &runner.Fake{Commands: map[string]runner.FakeResult{"docker compose version --format json": {Output: `{"version": "2.0.0"}`}}}

		got := health.CheckDockerComposeMinVersion(ctx, runner, "2.0.0")
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("accepts Docker Compose newer than the minimum version", func(t *testing.T) {
		runner := &runner.Fake{Commands: map[string]runner.FakeResult{"docker compose version --format json": {Output: `{"version": "5.2.0"}`}}}

		got := health.CheckDockerComposeMinVersion(ctx, runner, "2.0.0")
		var want *health.CheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("returns an upgrade fix when Docker Compose is too old", func(t *testing.T) {
		runner := &runner.Fake{Commands: map[string]runner.FakeResult{"docker compose version --format json": {Output: `{"version": "v1.9.0"}`}}}

		got := health.CheckDockerComposeMinVersion(ctx, runner, "2.0.0")
		want := &health.CheckFailure{
			Message: "installed docker compose version v1.9.0 is older than required version 2.0.0",
			Fix:     &health.Fix{Description: "Upgrade Docker Compose to version 2.0.0 or later. See https://github.com/arm/topo#install-a-container-engine"},
		}

		assert.Equal(t, want, got)
	})
}
