package health_test

import (
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestEvaluatedHealthCheck(t *testing.T) {
	t.Run("Report", func(t *testing.T) {
		t.Run("projects missing target severity without changing the shared result", func(t *testing.T) {
			dependency := health.EvaluatedDependency{
				ID:    health.DependencyIDTargetSpecified,
				Label: "Target specified",
				Result: health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityError,
					Message:  "target not specified",
					Fix:      &health.Fix{Description: "Choose a target"},
				}},
			}
			evaluated := health.EvaluatedHealthCheck{
				Deployment:       health.EvaluatedReadinessCheck{Target: []health.EvaluatedDependency{dependency}},
				ProjectDiscovery: health.EvaluatedReadinessCheck{Target: []health.EvaluatedDependency{dependency}},
			}

			got := evaluated.Report(health.TargetDetails{})

			assert.Equal(t, []health.DependencyReport{{
				ID: health.DependencyIDTargetSpecified, Name: "Target specified",
				Status: health.CheckStatusError, Value: "target not specified",
				Fix: &health.Fix{Description: "Choose a target"},
			}}, got.Deployment.Target)
			assert.Equal(t, []health.DependencyReport{{
				ID: health.DependencyIDTargetSpecified, Name: "Target specified",
				Status: health.CheckStatusWarning, Value: "target not specified; cannot calculate project compatibility",
				Fix: &health.Fix{Description: "Choose a target"},
			}}, got.ProjectDiscovery.Target)
			assert.Equal(t, health.SeverityError, dependency.Result.Failure.Severity)
		})

		t.Run("hides successful selection and local access", func(t *testing.T) {
			evaluated := health.EvaluatedHealthCheck{
				Deployment: health.EvaluatedReadinessCheck{Target: []health.EvaluatedDependency{
					{ID: health.DependencyIDTargetSpecified, Label: "Target specified"},
					{ID: health.DependencyIDConnectivity, Label: "Target access"},
					{Label: "Hardware Info", Result: health.DependencyCheckResult{SuccessValue: "lscpu"}},
				}},
			}

			got := evaluated.Report(health.TargetDetails{Destination: "localhost", IsLocalhost: true})

			assert.Equal(t, []health.DependencyReport{{
				Name: "Hardware Info", Status: health.CheckStatusOK, Value: "lscpu",
			}}, got.Deployment.Target)
		})

		t.Run("retains remote access results", func(t *testing.T) {
			evaluated := health.EvaluatedHealthCheck{
				Deployment: health.EvaluatedReadinessCheck{Target: []health.EvaluatedDependency{
					{ID: health.DependencyIDTargetSpecified, Label: "Target specified"},
					{ID: health.DependencyIDConnectivity, Label: "Target access", Result: health.DependencyCheckResult{SuccessValue: "user@example.com"}},
				}},
			}

			got := evaluated.Report(health.TargetDetails{Destination: "user@example.com"})

			assert.Equal(t, []health.DependencyReport{{
				ID: health.DependencyIDConnectivity, Name: "Target access", Status: health.CheckStatusOK, Value: "user@example.com",
			}}, got.Deployment.Target)
		})
	})
}

func TestToDependencyReport(t *testing.T) {
	t.Run("returns successful dependency result", func(t *testing.T) {
		status := health.EvaluatedDependency{
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
		status := health.EvaluatedDependency{
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
		status := health.EvaluatedDependency{
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
		status := health.EvaluatedDependency{
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
		status := health.EvaluatedDependency{
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
