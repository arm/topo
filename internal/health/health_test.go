package health_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("when no remoteproc devices are found, SubsystemDriver health check an info message", func(t *testing.T) {
		ts := health.Status{}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusInfo, got.SubsystemDriver.Status)
		assert.Equal(t, "no remoteproc devices found", got.SubsystemDriver.Value)
	})

	t.Run("when remoteproc probe fails, SubsystemDriver reports the error", func(t *testing.T) {
		ts := health.Status{
			Hardware: health.HardwareProfile{
				Err: fmt.Errorf("timed out"),
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.SubsystemDriver.Status)
		assert.Equal(t, "timed out", got.SubsystemDriver.Value)
	})

	t.Run("when remoteproc devices are found, SubsystemDriver status is ok and includes device names", func(t *testing.T) {
		ts := health.Status{
			Hardware: health.HardwareProfile{
				RemoteProcessors: []probe.RemoteProcessor{{Name: "m4_0"}, {Name: "m4_1"}},
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusOK, got.SubsystemDriver.Status)
		assert.Equal(t, "m4_0, m4_1", got.SubsystemDriver.Value)
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
				Destination: ssh.NewDestination("user@my-target"),
				Error:       probe.ErrHostKeyChanged,
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.Connectivity.Status)
		assert.Equal(t, "Remove the old SSH host key from known_hosts, then retry", got.Connectivity.Fix.Description)
		assert.Equal(t, "ssh-keygen -R my-target", got.Connectivity.Fix.Command)
	})
}

func TestHostReport(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		t.Run("nil dependencies are [] not null", func(t *testing.T) {
			tr := health.HostReport{Dependencies: nil}

			b, err := json.Marshal(tr)

			require.NoError(t, err)
			want := `{ "dependencies": [] }`
			assert.JSONEq(t, want, string(b))
		})

		t.Run("omits command when fix has no command", func(t *testing.T) {
			tr := health.HostReport{Dependencies: []health.HealthCheck{
				{
					Name:   "Container Engine",
					Status: health.CheckStatusError,
					Value:  "permission denied",
					Fix: &health.Fix{
						Description: "Ensure current user can run docker commands",
					},
				},
			}}

			b, err := json.Marshal(tr)

			require.NoError(t, err)
			want := `{
				"dependencies": [
					{
						"name": "Container Engine",
						"status": "error",
						"value": "permission denied",
						"fix": {
							"description": "Ensure current user can run docker commands"
						}
					}
				]
			}`
			assert.JSONEq(t, want, string(b))
		})
	})
}

func TestTargetReport(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		t.Run("nil dependencies are [] not null", func(t *testing.T) {
			tr := health.TargetReport{Dependencies: nil}

			b, err := json.Marshal(tr)

			require.NoError(t, err)
			var result map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(b, &result))
			assert.JSONEq(t, `[]`, string(result["dependencies"]))
		})
	})
}

func testDependencyReporting(t *testing.T, extract func([]health.DependencyStatus) []health.HealthCheck) {
	t.Helper()

	t.Run("when a dependency is not installed, health check reports error", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{Dependency: health.Dependency{Binary: "whatever", Label: "Rube Goldberg"}, Error: fmt.Errorf("whatever not found on path")},
		}

		got := extract(statuses)

		assert.Equal(t, []health.HealthCheck{
			{Name: "Rube Goldberg", Status: health.CheckStatusError, Value: "whatever not found on path"},
		}, got)
	})

	t.Run("when a dependency has a warning error, health check reports warning", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{Dependency: health.Dependency{Binary: "remoteproc-runtime", Label: "Remoteproc Runtime"}, Error: health.WarningError{Err: fmt.Errorf("remoteproc-runtime not found on path")}},
		}

		got := extract(statuses)

		assert.Equal(t, []health.HealthCheck{
			{Name: "Remoteproc Runtime", Status: health.CheckStatusWarning, Value: "remoteproc-runtime not found on path"},
		}, got)
	})

	t.Run("propagates Fix from DependencyStatus to HealthCheck", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Binary: "pizza", Label: "Food"},
				Error:      health.WarningError{Err: errors.New("not enough pineapple")},
				Fix: &health.Fix{
					Description: "add more pineapple",
					Command:     "pizza --pineapple",
				},
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
