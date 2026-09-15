package health_test

import (
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestGenerateHostReport(t *testing.T) {
	testDependencyReporting(t, func(statuses []health.DependencyStatus) []health.HealthCheck {
		return health.GenerateHostReport(statuses).Dependencies
	})
}

func TestGenerateTargetReport(t *testing.T) {
	testDependencyReporting(t, func(statuses []health.DependencyStatus) []health.HealthCheck {
		return health.GenerateTargetReport(health.Status{Dependencies: statuses}).Dependencies
	})
}

func testDependencyReporting(t *testing.T, extract func([]health.DependencyStatus) []health.HealthCheck) {
	t.Helper()

	t.Run("when a dependency has a successful result, health check reports its value", func(t *testing.T) {
		statuses := []health.DependencyStatus{{
			Dependency: health.Dependency{Label: "Container Engine"},
			Result:     health.DependencyCheckResult{SuccessValue: "docker"},
		}}

		got := extract(statuses)

		assert.Equal(t, []health.HealthCheck{{Name: "Container Engine", Status: health.CheckStatusOK, Value: "docker"}}, got)
	})

	t.Run("when a dependency has an error result, health check reports error", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Label: "Rube Goldberg"},
				Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityError,
					Message:  "whatever not found on path",
				}},
			},
		}

		got := extract(statuses)

		assert.Equal(t, []health.HealthCheck{
			{Name: "Rube Goldberg", Status: health.CheckStatusError, Value: "whatever not found on path"},
		}, got)
	})

	t.Run("when a dependency has a warning result, health check reports warning", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Label: "Remoteproc Runtime"},
				Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityWarning,
					Message:  "remoteproc-runtime not found on path",
				}},
			},
		}

		got := extract(statuses)

		assert.Equal(t, []health.HealthCheck{
			{Name: "Remoteproc Runtime", Status: health.CheckStatusWarning, Value: "remoteproc-runtime not found on path"},
		}, got)
	})

	t.Run("propagates Fix from CheckFailure to HealthCheck", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Label: "Food"},
				Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityWarning,
					Message:  "not enough pineapple",
					Fix: &health.Fix{
						Description: "add more pineapple",
						Command:     "pizza --pineapple",
					},
				}},
			},
		}

		got := extract(statuses)

		want := []health.HealthCheck{
			{
				Name:   "Food",
				Status: health.CheckStatusWarning,
				Value:  "not enough pineapple",
				Fix: &health.Fix{
					Description: "add more pineapple",
					Command:     "pizza --pineapple",
				},
			},
		}
		assert.Equal(t, want, got)
	})
}
