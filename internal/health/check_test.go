package health_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
)

func TestBinaryExists(t *testing.T) {
	t.Run("wraps error as WarningError when severity is warning", func(t *testing.T) {
		check := health.BinaryExists{
			Severity: health.SeverityWarning,
		}
		dependency := health.Dependency{Binary: "nonexistent"}
		runner := &runner.Fake{}
		ctx := context.Background()

		_, err := check.Apply(ctx, runner, dependency)

		wantErr := health.WarningError{Err: runner.BinaryExists(ctx, dependency.Binary)}
		assert.Equal(t, wantErr, err)
	})
}
