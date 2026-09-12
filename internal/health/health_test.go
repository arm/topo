package health_test

import (
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/ssh"
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

	t.Run("when the target has a connection error, Connectivity status reports error", func(t *testing.T) {
		ts := health.Status{Connection: health.ConnectionStatus{Error: assert.AnError}}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.Connectivity.Status)
		assert.Equal(t, assert.AnError.Error(), got.Connectivity.Value)
	})

	t.Run("when the target has no connection error, Connectivity status is ok", func(t *testing.T) {
		ts := health.Status{}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusOK, got.Connectivity.Status)
	})

	t.Run("when authentication fails, Connectivity includes a setup-keys fix", func(t *testing.T) {
		ts := health.Status{
			Connection: health.ConnectionStatus{
				Destination: ssh.NewDestination("user@my-target"),
				Error:       probe.ErrAuthFailed,
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.Connectivity.Status)
		assert.Equal(t, "Configure SSH keys on remote target", got.Connectivity.Fix.Description)
		assert.Equal(t, "topo setup-keys --target ssh://user@my-target", got.Connectivity.Fix.Command)
	})

	t.Run("when too many authentication failures occur, Connectivity includes a setup-keys fix", func(t *testing.T) {
		ts := health.Status{
			Connection: health.ConnectionStatus{
				Destination: ssh.NewDestination("user@my-target"),
				Error:       probe.ErrTooManyAuthFails,
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.Connectivity.Status)
		assert.Equal(t, "Configure SSH keys on remote target", got.Connectivity.Fix.Description)
		assert.Equal(t, "topo setup-keys --target ssh://user@my-target", got.Connectivity.Fix.Command)
	})

	t.Run("when host key is new, Connectivity includes an accept-new-host-keys fix", func(t *testing.T) {
		ts := health.Status{
			Connection: health.ConnectionStatus{
				Destination: ssh.NewDestination("user@my-target"),
				Error:       probe.ErrHostKeyUnknown,
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.Connectivity.Status)
		assert.Equal(t, "Trust the target's SSH host key", got.Connectivity.Fix.Description)
		assert.Equal(t, "topo health --target ssh://user@my-target --accept-new-host-keys", got.Connectivity.Fix.Command)
	})

	t.Run("when host key has changed, Connectivity includes a known_hosts fix", func(t *testing.T) {
		ts := health.Status{
			Connection: health.ConnectionStatus{
				Destination: ssh.NewDestination("ssh://user@my-target:2222"),
				Error:       probe.ErrHostKeyChanged,
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.Connectivity.Status)
		assert.Equal(t, "Remove the old SSH host key from known_hosts, then retry", got.Connectivity.Fix.Description)
		assert.Equal(t, "ssh-keygen -R '[my-target]:2222'", got.Connectivity.Fix.Command)
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
