package health_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	t.Run("ids are unique across all dependencies", func(t *testing.T) {
		hostDeps := health.HostRequiredDependencies(false)
		targetDeps := health.TargetRequiredDependencies(ssh.NewDestination("whatever"), false)

		ids := make([]health.DependencyID, 0, len(hostDeps)+len(targetDeps))
		for _, dep := range slices.Concat(hostDeps, targetDeps) {
			require.NotEmpty(t, dep.ID, "%#v has empty id", dep)
			require.NotContains(t, ids, dep.ID)
			ids = append(ids, dep.ID)
		}
	})

	t.Run("host dependencies", func(t *testing.T) {
		t.Run("prerequisites are fulfillable", func(t *testing.T) {
			deps := health.HostRequiredDependencies(false)
			ids := make([]health.DependencyID, 0, len(deps))
			for _, dep := range deps {
				ids = append(ids, dep.ID)
				for _, prereq := range dep.Prerequisites {
					require.Contains(t, ids, prereq)
				}
			}
		})
	})

	t.Run("target dependencies", func(t *testing.T) {
		t.Run("remote target dependencies require connectivity", func(t *testing.T) {
			deps := health.TargetRequiredDependencies(ssh.NewDestination("user@my-target"), false)

			for _, dep := range deps {
				if dep.ID == health.DependencyIDConnectivity {
					continue
				}
				assert.Contains(t, dep.Prerequisites, health.DependencyIDConnectivity, dep.ID)
			}
		})

		t.Run("prerequisites are fulfillable", func(t *testing.T) {
			deps := health.TargetRequiredDependencies(ssh.NewDestination("does-not-matter-for-this-test"), false)
			ids := make([]health.DependencyID, 0, len(deps))
			for _, dep := range deps {
				ids = append(ids, dep.ID)
				for _, prereq := range dep.Prerequisites {
					require.Contains(t, ids, prereq)
				}
			}
		})
	})
}

func TestPerformChecks(t *testing.T) {
	t.Run("dependency status reflects the result of running the check", func(t *testing.T) {
		t.Run("when check passes", func(t *testing.T) {
			dep := health.Dependency{Label: "bar", Check: passingCheck}
			deps := []health.Dependency{dep}

			got := health.PerformChecks(context.Background(), deps)

			require.Len(t, got, 1)
			assert.Equal(t, dep.ID, got[0].Dependency.ID)
			assert.Nil(t, got[0].Result.Failure)
		})

		t.Run("when a check fails", func(t *testing.T) {
			dep := health.Dependency{Label: "bar", Check: failingCheck}
			deps := []health.Dependency{dep}

			got := health.PerformChecks(context.Background(), deps)

			wantResult := failingCheck(context.Background())
			require.Len(t, got, 1)
			assert.Equal(t, dep.ID, got[0].Dependency.ID)
			assert.Equal(t, wantResult, got[0].Result)
		})
	})

	t.Run("prerequisites", func(t *testing.T) {
		t.Run("omits dependency when any of its prerequisites is failing", func(t *testing.T) {
			pineapple := health.Dependency{
				ID:    health.DependencyID("pineapple"),
				Check: passingCheck,
			}
			cheese := health.Dependency{
				ID:    health.DependencyID("cheese"),
				Check: failingCheck,
			}
			pizzaWhichShouldBeOmitted := health.Dependency{
				ID:            "pizza",
				Prerequisites: []health.DependencyID{pineapple.ID, cheese.ID},
			}
			deps := []health.Dependency{
				pineapple,
				cheese,
				pizzaWhichShouldBeOmitted,
			}

			got := health.PerformChecks(context.Background(), deps)

			assert.Len(t, got, 2)
			assert.NotContains(t, got, health.DependencyStatus{Dependency: pizzaWhichShouldBeOmitted})
		})

		t.Run("checks dependency when all of its prerequisites are passing", func(t *testing.T) {
			vader := health.Dependency{
				ID:    health.DependencyID("vader"),
				Check: passingCheck,
			}
			luke := health.Dependency{
				ID:            "luke",
				Prerequisites: []health.DependencyID{vader.ID},
			}
			deps := []health.Dependency{vader, luke}

			got := health.PerformChecks(context.Background(), deps)

			require.Len(t, got, 2)
			assert.Equal(t, vader.ID, got[0].Dependency.ID)
			assert.Equal(t, luke.ID, got[1].Dependency.ID)
		})
	})
}

func TestRemoteprocDependency(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		buildRunnerWithRemoteProcs := func(names []string) runner.Runner {
			return &runner.Fake{Commands: map[string]runner.FakeResult{
				"cat /sys/class/remoteproc/*/name": {Output: strings.Join(names, "\n")},
			}}
		}

		t.Run("fails when no remoteproc devices are found", func(t *testing.T) {
			r := buildRunnerWithRemoteProcs(nil)
			d := health.NewRemoteprocDependency(r)

			got := d.Check(context.Background())

			want := health.DependencyCheckResult{
				Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityInfo,
					Message:  "no remoteproc devices found",
				},
			}
			assert.Equal(t, want, got)
		})

		t.Run("fails when remoteproc probe fails", func(t *testing.T) {
			r := &runner.Fake{Commands: map[string]runner.FakeResult{
				"cat /sys/class/remoteproc/*/name": {Err: runner.ErrTimeout},
			}}
			d := health.NewRemoteprocDependency(r)

			got := d.Check(context.Background())

			want := health.DependencyCheckResult{
				Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityError,
					Message:  "timed out",
				},
			}
			assert.Equal(t, want, got)
		})

		t.Run("reports remoteproc device names", func(t *testing.T) {
			r := buildRunnerWithRemoteProcs([]string{"m4_0", "m4_1"})
			d := health.NewRemoteprocDependency(r)

			got := d.Check(context.Background())

			assert.Equal(t, health.DependencyCheckResult{SuccessValue: "m4_0, m4_1"}, got)
		})
	})
}

func TestRemoteprocRuntimeDependency(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		t.Run("includes an install fix with the target", func(t *testing.T) {
			dep := health.NewRemoteprocRuntimeDependency(ssh.NewDestination("user@my-target"), &runner.Fake{})

			result := dep.Check(context.Background())

			assert.Equal(t, &health.DependencyCheckFailure{
				Severity: health.SeverityWarning,
				Message:  `"remoteproc-runtime" not found in $PATH`,
				Fix: &health.Fix{
					Description: "Install the Remoteproc Runtime",
					Command:     "topo install remoteproc-runtime --target ssh://user@my-target",
				},
			}, result.Failure)
		})
	})
}

func passingCheck(_ context.Context) health.DependencyCheckResult {
	return health.DependencyCheckResult{SuccessValue: "passed"}
}

func failingCheck(_ context.Context) health.DependencyCheckResult {
	return health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
		Severity: health.SeverityError,
		Message:  "very broken",
		Fix:      &health.Fix{Description: "fix me please", Command: "echo fixed"},
	}}
}
