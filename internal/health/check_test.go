package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
)

func TestCheckOpenSSHAvailable(t *testing.T) {
	ctx := context.Background()
	t.Run("accepts OpenSSH", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Stderr: "OpenSSH_9.9p1, OpenSSL 3.4.0"}}}

		got := health.CheckOpenSSHAvailable(ctx, r, "ssh")
		var want *health.DependencyCheckFailure

		assert.Equal(t, want, got)
	})

	t.Run("rejects another SSH implementation", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Stderr: "Dropbear v2025.88"}}}

		got := health.CheckOpenSSHAvailable(ctx, r, "ssh")
		want := &health.DependencyCheckFailure{
			Message: `"ssh" does not resolve to OpenSSH: Dropbear v2025.88`,
			Fix:     &health.Fix{Description: "Install OpenSSH and ensure its ssh executable is first on PATH"},
		}

		assert.Equal(t, want, got)
	})

	t.Run("fails when the version cannot be checked", func(t *testing.T) {
		versionErr := errors.New("version check failed")
		r := &runner.Fake{Commands: map[string]runner.FakeResult{"ssh -V": {Err: versionErr}}}

		got := health.CheckOpenSSHAvailable(ctx, r, "ssh")
		want := &health.DependencyCheckFailure{Message: versionErr.Error()}

		assert.Equal(t, want, got)
	})
}
