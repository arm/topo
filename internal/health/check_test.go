package health_test

import (
	"context"
	"fmt"
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
