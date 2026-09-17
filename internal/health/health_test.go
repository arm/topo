package health_test

import (
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestToDependencyReport(t *testing.T) {
	t.Run("returns successful dependency result", func(t *testing.T) {
		status := health.DependencyStatus{
			ID:     "docker",
			Label:  "Container Engine",
			Result: health.DependencyCheckResult{SuccessValue: "docker"},
		}

		got := health.ToDependencyReport(status)

		want := health.DependencyReport{
			ID:     "docker",
			Name:   "Container Engine",
			Status: health.CheckStatusOK,
			Value:  "docker",
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns error dependency result", func(t *testing.T) {
		status := health.DependencyStatus{
			Label: "Rube Goldberg",
			Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
				Severity: health.SeverityError,
				Message:  "whatever not found on path",
			}},
		}

		got := health.ToDependencyReport(status)

		want := health.DependencyReport{
			Name:   "Rube Goldberg",
			Status: health.CheckStatusError,
			Value:  "whatever not found on path",
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns warning dependency result", func(t *testing.T) {
		status := health.DependencyStatus{
			Label: "Remoteproc Runtime",
			Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
				Severity: health.SeverityWarning,
				Message:  "remoteproc-runtime not found on path",
			}},
		}

		got := health.ToDependencyReport(status)

		want := health.DependencyReport{
			Name:   "Remoteproc Runtime",
			Status: health.CheckStatusWarning,
			Value:  "remoteproc-runtime not found on path",
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns informational dependency result", func(t *testing.T) {
		status := health.DependencyStatus{
			Label: "Remoteproc Runtime",
			Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
				Severity: health.SeverityInfo,
				Message:  "no remoteproc devices found",
			}},
		}

		got := health.ToDependencyReport(status)

		want := health.DependencyReport{
			Name:   "Remoteproc Runtime",
			Status: health.CheckStatusInfo,
			Value:  "no remoteproc devices found",
		}
		assert.Equal(t, want, got)
	})

	t.Run("propagates fix from failed dependency", func(t *testing.T) {
		status := health.DependencyStatus{
			Label: "Food",
			Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
				Severity: health.SeverityWarning,
				Message:  "not enough pineapple",
				Fix: &health.Fix{
					Description: "add more pineapple",
					Command:     "pizza --pineapple",
				},
			}},
		}

		got := health.ToDependencyReport(status)

		want := health.DependencyReport{
			Name:   "Food",
			Status: health.CheckStatusWarning,
			Value:  "not enough pineapple",
			Fix: &health.Fix{
				Description: "add more pineapple",
				Command:     "pizza --pineapple",
			},
		}
		assert.Equal(t, want, got)
	})
}
