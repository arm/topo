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
	t.Run("when no dependencies are found, statuses show not installed", func(t *testing.T) {
		fooDependency := health.Dependency{Binary: "foo", Label: "bar", Checks: []health.Check{health.BinaryExists{}}}
		bazDependency := health.Dependency{Binary: "baz", Label: "qux", Checks: []health.Check{health.BinaryExists{}}}
		deps := []health.Dependency{fooDependency, bazDependency}
		runner := &runner.Fake{}

		got := health.PerformChecks(context.Background(), deps, runner)

		wantFoo := health.DependencyStatus{Dependency: fooDependency, Error: runner.BinaryExists(context.Background(), fooDependency.Binary)}
		wantBar := health.DependencyStatus{Dependency: bazDependency, Error: runner.BinaryExists(context.Background(), bazDependency.Binary)}
		want := []health.DependencyStatus{wantFoo, wantBar}
		assert.Equal(t, want, got)
	})

	t.Run("when a dependency is found, its status entry reflects that", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "baz", Label: "qux", Checks: []health.Check{health.BinaryExists{}}},
		}
		runner := &runner.Fake{
			Binaries: []string{"baz"},
		}

		got := health.PerformChecks(context.Background(), deps, runner)

		want := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Binary: "baz", Label: "qux", Checks: []health.Check{health.BinaryExists{}}},
				Error:      nil,
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("omits dependency when any of its software prerequisites are not installed", func(t *testing.T) {
		pineapple := health.Dependency{
			ID:     health.DependencyID("pineapple"),
			Binary: "pineapple",
			Checks: []health.Check{health.BinaryExists{}},
		}
		cheese := health.Dependency{
			ID:     health.DependencyID("cheese"),
			Binary: "cheese",
			Checks: []health.Check{health.BinaryExists{}},
		}
		deps := []health.Dependency{
			pineapple,
			cheese,
			{
				ID:                    "pizza",
				SoftwarePrerequisites: []health.DependencyID{pineapple.ID, cheese.ID},
			},
		}
		runner := &runner.Fake{Binaries: []string{pineapple.Binary}}

		got := health.PerformChecks(context.Background(), deps, runner)

		want := []health.DependencyStatus{
			{Dependency: pineapple},
			{Dependency: cheese, Error: runner.BinaryExists(context.Background(), cheese.Binary)},
		}
		assert.Equal(t, want, got)
	})

	t.Run("checks dependency when all of its software prerequisites are installed", func(t *testing.T) {
		vader := health.Dependency{
			ID:     health.DependencyID("vader"),
			Binary: "vader",
			Checks: []health.Check{health.BinaryExists{}},
		}
		luke := health.Dependency{
			ID:                    "luke",
			SoftwarePrerequisites: []health.DependencyID{vader.ID},
		}
		deps := []health.Dependency{vader, luke}
		runner := &runner.Fake{Binaries: []string{vader.Binary}}

		got := health.PerformChecks(context.Background(), deps, runner)

		want := []health.DependencyStatus{
			{Dependency: vader},
			{Dependency: luke},
		}
		assert.Equal(t, want, got)
	})

	t.Run("captures fix from failing check", func(t *testing.T) {
		dep := health.Dependency{
			Binary: "remoteproc-runtime",
			Label:  "Remoteproc Runtime",
			Checks: []health.Check{
				health.BinaryExists{
					Severity: health.SeverityWarning,
					Fix: &health.Fix{
						Description: "Install the Remoteproc Runtime",
						Command:     "topo install remoteproc-runtime",
					},
				},
			},
		}
		runner := &runner.Fake{}

		got := health.PerformChecks(context.Background(), []health.Dependency{dep}, runner)

		assert.Len(t, got, 1)
		want := &health.Fix{
			Description: "Install the Remoteproc Runtime",
			Command:     "topo install remoteproc-runtime",
		}
		assert.Equal(t, want, got[0].Fix)
	})

	t.Run("checks dependency with no SoftwarePrerequisites unconditionally", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "standalone", Label: "Tools", Checks: []health.Check{health.BinaryExists{}}},
		}
		runner := &runner.Fake{
			Binaries: []string{"standalone"},
		}

		got := health.PerformChecks(context.Background(), deps, runner)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Binary: "standalone", Label: "Tools", Checks: []health.Check{health.BinaryExists{}}}, Error: nil},
		}
		assert.Equal(t, want, got)
	})

	t.Run("captures failure from a command successful check and verifies that arguments are passed correctly", func(t *testing.T) {
		dep := health.Dependency{
			Binary: "potatoes",
			Label:  "Air Fryer Engine",
			Checks: []health.Check{health.BinaryExists{}, health.CommandSuccessful{
				Cmd: "potatoes --cook-well",
				Fix: &health.Fix{Description: "Ensure current user can run the potatoe cooker"},
			}},
		}
		runner := &runner.Fake{
			Binaries: []string{"potatoes"},
			Commands: map[string]runner.FakeResult{
				"potatoes --cook-well": {
					Err: errors.New("permission denied"),
				},
			},
		}

		got := health.PerformChecks(context.Background(), []health.Dependency{dep}, runner)

		want := []health.DependencyStatus{
			{
				Dependency: dep,
				Error:      errors.New("permission denied"),
				Fix:        &health.Fix{Description: "Ensure current user can run the potatoe cooker"},
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("timeout skips unverified prerequisite dependents", func(t *testing.T) {
		dockerDep := health.Dependency{
			Binary: "docker",
			Label:  "Container Engine",
			ID:     health.DependencyID("docker"),
			Checks: []health.Check{health.BinaryExists{}},
		}
		runtimeDep := health.Dependency{
			Binary:                "runtime",
			Label:                 "Runtime",
			SoftwarePrerequisites: []health.DependencyID{dockerDep.ID},
			Checks:                []health.Check{health.BinaryExists{}},
		}
		standaloneDep := health.Dependency{
			Binary: "lscpu",
			Label:  "Hardware Info",
			Checks: []health.Check{health.BinaryExists{}},
		}
		r := &runner.Fake{
			BinaryExistsErr: map[string]error{dockerDep.Binary: runner.ErrTimeout},
			Binaries:        []string{standaloneDep.Binary},
		}

		got := health.PerformChecks(context.Background(), []health.Dependency{dockerDep, runtimeDep, standaloneDep}, r)

		assert.Len(t, got, 2)
		assert.Equal(t, "Container Engine", got[0].Dependency.Label)
		assert.ErrorIs(t, got[0].Error, runner.ErrTimeout)
		assert.Equal(t, "Hardware Info", got[1].Dependency.Label)
		assert.NoError(t, got[1].Error)
	})

	t.Run("timeout on warning severity check is not wrapped as WarningError", func(t *testing.T) {
		dep := health.Dependency{
			Binary: "optional-tool",
			Label:  "Optional",
			Checks: []health.Check{health.BinaryExists{Severity: health.SeverityWarning}},
		}
		r := &runner.Fake{
			BinaryExistsErr: map[string]error{"optional-tool": runner.ErrTimeout},
		}

		got := health.PerformChecks(context.Background(), []health.Dependency{dep}, r)

		assert.Len(t, got, 1)
		assert.ErrorIs(t, got[0].Error, runner.ErrTimeout)
		_, isWarning := got[0].Error.(health.WarningError)
		assert.False(t, isWarning)
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
