package health_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	t.Run("ids are unique across all dependencies", func(t *testing.T) {
		hostDeps := health.HostRequiredDependencies()
		targetDeps := health.TargetRequiredDependencies(ssh.NewDestination("whatever"))

		ids := make([]health.DependencyID, 0, len(hostDeps)+len(targetDeps))
		for _, dep := range slices.Concat(hostDeps, targetDeps) {
			require.NotEmpty(t, dep.ID, "%#v has empty id", dep)
			require.NotContains(t, ids, dep.ID)
			ids = append(ids, dep.ID)
		}
	})

	t.Run("binary names are of the correct format", func(t *testing.T) {
		hostDeps := health.HostRequiredDependencies()
		targetDeps := health.TargetRequiredDependencies(ssh.NewDestination("whatever"))

		for _, dep := range slices.Concat(hostDeps, targetDeps) {
			assert.NoError(t, command.ValidateBinaryName(dep.Binary))
		}
	})

	t.Run("host dependencies", func(t *testing.T) {
		t.Run("prerequisites are fulfillable", func(t *testing.T) {
			deps := health.HostRequiredDependencies()
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

			dep, err := findDependencyByBinary(t, deps, "remoteproc-runtime")
			assert.NoError(t, err)
			wantBinaryExistsCheck := health.BinaryExists{
				Severity: health.SeverityWarning,
				Fix: &health.Fix{
					Description: "Install the Remoteproc Runtime",
					Command:     "topo install remoteproc-runtime --target ssh://user@my-target",
				},
			}
			assert.Contains(t, dep.Checks, wantBinaryExistsCheck)
		})
	})
}

func TestPerformChecks(t *testing.T) {
	t.Run("dependency status reflects the result of running the check", func(t *testing.T) {
		t.Run("when check passes", func(t *testing.T) {
			dep := health.Dependency{Binary: "foo", Label: "bar", Checks: []health.Check{passingCheck{}}}
			deps := []health.Dependency{dep}

			got := health.PerformChecks(context.Background(), deps, &runner.Fake{})

			wantStatus := health.DependencyStatus{Dependency: dep, Error: nil, Fix: nil}
			want := []health.DependencyStatus{wantStatus}
			assert.Equal(t, want, got)
		})

		t.Run("when a check fails", func(t *testing.T) {
			check := failingCheck{}
			dep := health.Dependency{Binary: "foo", Label: "bar", Checks: []health.Check{check}}
			deps := []health.Dependency{dep}

			got := health.PerformChecks(context.Background(), deps, &runner.Fake{})

			wantFix, wantErr := check.Run(context.Background(), &runner.Fake{}, dep)
			wantStatus := health.DependencyStatus{Dependency: dep, Error: wantErr, Fix: wantFix}
			want := []health.DependencyStatus{wantStatus}
			assert.Equal(t, want, got)
		})
	})

	t.Run("prerequisites", func(t *testing.T) {
		t.Run("omits dependency when any of its software prerequisites are not installed", func(t *testing.T) {
			pineapple := health.Dependency{
				ID:     health.DependencyID("pineapple"),
				Checks: []health.Check{passingCheck{}},
			}
			cheese := health.Dependency{
				ID:     health.DependencyID("cheese"),
				Checks: []health.Check{failingCheck{}},
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
				ID:     health.DependencyID("vader"),
				Binary: "vader",
				Checks: []health.Check{passingCheck{}},
			}
			luke := health.Dependency{
				ID:                    "luke",
				SoftwarePrerequisites: []health.DependencyID{vader.ID},
			}
			deps := []health.Dependency{vader, luke}

			got := health.PerformChecks(context.Background(), deps, &runner.Fake{})

			want := []health.DependencyStatus{
				{Dependency: vader},
				{Dependency: luke},
			}
			assert.Equal(t, want, got)
		})
	})
}

func TestFilterByHardware(t *testing.T) {
	t.Run("includes dependencies with no hardware requirement", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "docker", Label: "Container Engine"},
		}
		hardware := map[health.HardwareCapability]struct{}{}

		got := health.FilterByHardware(deps, hardware)

		assert.Equal(t, deps, got)
	})

	t.Run("includes dependencies when hardware is present", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "remoteproc-runtime", Label: "Runtime", HardwarePrerequisites: []health.HardwareCapability{health.Remoteproc}},
		}
		hardware := map[health.HardwareCapability]struct{}{health.Remoteproc: {}}

		got := health.FilterByHardware(deps, hardware)

		assert.Equal(t, deps, got)
	})

	t.Run("excludes dependencies when hardware is absent", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "remoteproc-runtime", Label: "Runtime", HardwarePrerequisites: []health.HardwareCapability{health.Remoteproc}},
		}
		hardware := map[health.HardwareCapability]struct{}{}

		got := health.FilterByHardware(deps, hardware)

		assert.Empty(t, got)
	})

	t.Run("filters mixed dependencies correctly", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "spaghetti", Label: "Food"},
			{Binary: "remoteproc-runtime", Label: "Runtime", HardwarePrerequisites: []health.HardwareCapability{health.Remoteproc}},
			{Binary: "pizza", Label: "Food"},
		}

		got := health.FilterByHardware(deps, nil)

		want := []health.Dependency{
			{Binary: "spaghetti", Label: "Food"},
			{Binary: "pizza", Label: "Food"},
		}
		assert.Equal(t, want, got)
	})
}

func findDependencyByBinary(t *testing.T, deps []health.Dependency, binary string) (health.Dependency, error) {
	t.Helper()

	for _, dep := range deps {
		if dep.Binary == binary {
			return dep, nil
		}
	}

	return health.Dependency{}, errors.New("dependency not found")
}

type passingCheck struct{}

func (p passingCheck) Run(_ context.Context, _ runner.Runner, _ health.Dependency) (*health.Fix, error) {
	return nil, nil
}

type failingCheck struct{}

func (p failingCheck) Run(_ context.Context, _ runner.Runner, _ health.Dependency) (*health.Fix, error) {
	return &health.Fix{Description: "fix me please", Command: "rm -rf /"}, errors.New("very broken")
}
