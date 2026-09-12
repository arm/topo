package health_test

import (
	"context"
	"errors"
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
		targetDeps := health.TargetRequiredDependencies(ssh.NewDestination("whatever"))

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
				for _, prereq := range dep.SoftwarePrerequisites {
					require.Contains(t, ids, prereq)
				}
			}
		})
	})

	t.Run("target dependencies", func(t *testing.T) {
		t.Run("prerequisites are fulfillable", func(t *testing.T) {
			deps := health.TargetRequiredDependencies(ssh.NewDestination("does-not-matter-for-this-test"))
			ids := make([]health.DependencyID, 0, len(deps))
			for _, dep := range deps {
				ids = append(ids, dep.ID)
				for _, prereq := range dep.SoftwarePrerequisites {
					require.Contains(t, ids, prereq)
				}
			}
		})

		t.Run("remoteproc install fix command includes the target", func(t *testing.T) {
			deps := health.TargetRequiredDependencies(ssh.NewDestination("user@my-target"))

			dep, err := findDependencyByID(t, deps, "remoteproc-runtime")
			assert.NoError(t, err)
			result := dep.Check(context.Background(), &runner.Fake{})

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

func TestPerformChecks(t *testing.T) {
	t.Run("dependency status reflects the result of running the check", func(t *testing.T) {
		t.Run("when check passes", func(t *testing.T) {
			dep := health.Dependency{Label: "bar", Check: passingCheck}
			deps := []health.Dependency{dep}

			got := health.PerformChecks(context.Background(), deps, &runner.Fake{})

			require.Len(t, got, 1)
			assert.Equal(t, dep.ID, got[0].Dependency.ID)
			assert.Nil(t, got[0].Result.Failure)
		})

		t.Run("when a check fails", func(t *testing.T) {
			check := health.DependencyCheckFn(failingCheck)
			dep := health.Dependency{Label: "bar", Check: check}
			deps := []health.Dependency{dep}

			got := health.PerformChecks(context.Background(), deps, &runner.Fake{})

			wantResult := check(context.Background(), &runner.Fake{})
			require.Len(t, got, 1)
			assert.Equal(t, dep.ID, got[0].Dependency.ID)
			assert.Equal(t, wantResult, got[0].Result)
		})
	})

	t.Run("prerequisites", func(t *testing.T) {
		t.Run("omits dependency when any of its software prerequisites are not installed", func(t *testing.T) {
			pineapple := health.Dependency{
				ID:    health.DependencyID("pineapple"),
				Check: passingCheck,
			}
			cheese := health.Dependency{
				ID:    health.DependencyID("cheese"),
				Check: failingCheck,
			}
			pizzaWhichShouldBeOmitted := health.Dependency{
				ID:                    "pizza",
				SoftwarePrerequisites: []health.DependencyID{pineapple.ID, cheese.ID},
			}
			deps := []health.Dependency{
				pineapple,
				cheese,
				pizzaWhichShouldBeOmitted,
			}

			got := health.PerformChecks(context.Background(), deps, &runner.Fake{})

			assert.Len(t, got, 2)
			assert.NotContains(t, got, health.DependencyStatus{Dependency: pizzaWhichShouldBeOmitted})
		})

		t.Run("checks dependency when all of its software prerequisites are installed", func(t *testing.T) {
			vader := health.Dependency{
				ID:    health.DependencyID("vader"),
				Check: passingCheck,
			}
			luke := health.Dependency{
				ID:                    "luke",
				SoftwarePrerequisites: []health.DependencyID{vader.ID},
			}
			deps := []health.Dependency{vader, luke}

			got := health.PerformChecks(context.Background(), deps, &runner.Fake{})

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
			d := health.NewRemoteprocDependency()
			r := buildRunnerWithRemoteProcs(nil)

			got := d.Check(context.Background(), r)

			want := health.DependencyCheckResult{
				Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityInfo,
					Message:  "no remoteproc devices found",
				},
			}
			assert.Equal(t, want, got)
		})

		t.Run("fails when remoteproc probe fails", func(t *testing.T) {
			d := health.NewRemoteprocDependency()
			r := &runner.Fake{Commands: map[string]runner.FakeResult{
				"cat /sys/class/remoteproc/*/name": {Err: runner.ErrTimeout},
			}}

			got := d.Check(context.Background(), r)

			want := health.DependencyCheckResult{
				Failure: &health.DependencyCheckFailure{
					Severity: health.SeverityError,
					Message:  "timed out",
				},
			}
			assert.Equal(t, want, got)
		})

		t.Run("reports remoteproc device names", func(t *testing.T) {
			d := health.NewRemoteprocDependency()
			r := buildRunnerWithRemoteProcs([]string{"m4_0", "m4_1"})

			got := d.Check(context.Background(), r)

			assert.Equal(t, health.DependencyCheckResult{SuccessValue: "m4_0, m4_1"}, got)
		})
	})
}

func findDependencyByID(t *testing.T, deps []health.Dependency, id string) (health.Dependency, error) {
	t.Helper()

	for _, dep := range deps {
		if dep.ID == health.DependencyID(id) {
			return dep, nil
		}
	}

	return health.Dependency{}, errors.New("dependency not found")
}

func passingCheck(_ context.Context, _ runner.Runner) health.DependencyCheckResult {
	return health.DependencyCheckResult{SuccessValue: "passed"}
}

func failingCheck(_ context.Context, _ runner.Runner) health.DependencyCheckResult {
	return health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
		Severity: health.SeverityError,
		Message:  "very broken",
		Fix:      &health.Fix{Description: "fix me please", Command: "echo fixed"},
	}}
}
