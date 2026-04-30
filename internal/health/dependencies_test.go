package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
)

func TestDependencyFormat(t *testing.T) {
	t.Run("host dependencies are of the correct format", func(t *testing.T) {
		for _, dep := range health.HostRequiredDependencies {
			assert.NoError(t, command.ValidateBinaryName(dep.Binary))
		}
	})

	t.Run("target dependencies are of the correct format", func(t *testing.T) {
		for _, dep := range health.TargetRequiredDependencies {
			assert.NoError(t, command.ValidateBinaryName(dep.Binary))
		}
	})

	t.Run("target SoftwarePrerequisites reference valid dependencies", func(t *testing.T) {
		availableEnums := make(map[health.SoftwareDependency]bool)
		seenEnums := make(map[health.SoftwareDependency]string)

		t.Run("There are no duplicate SoftwareEnumID assignments", func(t *testing.T) {
			for _, dep := range health.TargetRequiredDependencies {
				if dep.SoftwareEnumID != health.UnsetSoftwareDependency {
					if existingDep, exists := seenEnums[dep.SoftwareEnumID]; exists {
						t.Errorf("Duplicate SoftwareEnumID %d assigned to both %q and %q", dep.SoftwareEnumID, existingDep, dep.Binary)
					}
					seenEnums[dep.SoftwareEnumID] = dep.Binary
					availableEnums[dep.SoftwareEnumID] = true
				}
			}
		})

		t.Run("all SoftwarePrerequisites reference valid SoftwareEnumID", func(t *testing.T) {
			for _, dep := range health.TargetRequiredDependencies {
				for _, prereq := range dep.SoftwarePrerequisites {
					assert.True(t, availableEnums[prereq], "%q has SoftwarePrerequisites %v which is not provided by any dependency's SoftwareEnumID", dep.Binary, prereq)
				}
			}
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

	t.Run("omits dependency when none of its SoftwarePrerequisites are installed", func(t *testing.T) {
		dockerDependecy := health.Dependency{Binary: "docker", Label: "Container Engine", Checks: []health.Check{health.BinaryExists{}}}
		deps := []health.Dependency{
			dockerDependecy,
			{Binary: "runtime", Label: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists{}}},
		}
		runner := &runner.Fake{
			Binaries: []string{"runtime"},
		}

		got := health.PerformChecks(context.Background(), deps, runner)

		wantDocker := health.DependencyStatus{Dependency: dockerDependecy, Error: runner.BinaryExists(context.Background(), dockerDependecy.Binary)}
		want := []health.DependencyStatus{wantDocker}
		assert.Equal(t, want, got)
	})

	t.Run("checks dependency when one of its SoftwarePrerequisites is installed", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "docker", Label: "Container Engine", SoftwareEnumID: health.Docker, Checks: []health.Check{health.BinaryExists{}}},
			{Binary: "runtime", Label: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists{}}},
		}
		runner := &runner.Fake{
			Binaries: []string{"runtime", "docker"},
		}

		got := health.PerformChecks(context.Background(), deps, runner)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Binary: "docker", Label: "Container Engine", SoftwareEnumID: health.Docker, Checks: []health.Check{health.BinaryExists{}}}, Error: nil},
			{Dependency: health.Dependency{Binary: "runtime", Label: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists{}}}, Error: nil},
		}
		assert.Equal(t, want, got)
	})

	t.Run("captures Fix from failing check", func(t *testing.T) {
		dep := health.Dependency{
			Binary: "vader.exe",
			Label:  "Sith",
			Checks: []health.Check{
				health.BinaryExists{
					Severity: health.SeverityWarning,
					Fix:      "turn Anakin into a bad man",
				},
			},
		}
		runner := &runner.Fake{}

		got := health.PerformChecks(context.Background(), []health.Dependency{dep}, runner)

		assert.Len(t, got, 1)
		assert.Equal(t, "turn Anakin into a bad man", got[0].Fix)
	})

	t.Run("captures FixCommand from failing check", func(t *testing.T) {
		dep := health.Dependency{
			Binary: "remoteproc-runtime",
			Label:  "Remoteproc Runtime",
			Checks: []health.Check{
				health.BinaryExists{
					Severity:   health.SeverityWarning,
					Fix:        "run `topo install remoteproc-runtime`",
					FixCommand: "topo install remoteproc-runtime",
				},
			},
		}
		runner := &runner.Fake{}

		got := health.PerformChecks(context.Background(), []health.Dependency{dep}, runner)

		assert.Len(t, got, 1)
		assert.Equal(t, "topo install remoteproc-runtime", got[0].FixCommand)
		assert.Equal(t, "run `topo install remoteproc-runtime`", got[0].Fix)
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
				Fix: "Ensure current user can run the potatoe cooker",
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
				Fix:        "Ensure current user can run the potatoe cooker",
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("timeout skips unverified prerequisite dependents", func(t *testing.T) {
		dockerDep := health.Dependency{
			Binary:         "docker",
			Label:          "Container Engine",
			SoftwareEnumID: health.Docker,
			Checks:         []health.Check{health.BinaryExists{}},
		}
		runtimeDep := health.Dependency{
			Binary:                "runtime",
			Label:                 "Runtime",
			SoftwarePrerequisites: []health.SoftwareDependency{health.Docker},
			Checks:                []health.Check{health.BinaryExists{}},
		}
		standaloneDep := health.Dependency{
			Binary: "lscpu",
			Label:  "Hardware Info",
			Checks: []health.Check{health.BinaryExists{}},
		}
		r := &runner.Fake{
			BinaryExistsErr: map[string]error{"docker": runner.ErrTimeout},
			Binaries:        []string{"lscpu"},
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
			{Binary: "remoteproc-runtime", Label: "Runtime", HardwarePrerequisite: []health.HardwareCapability{health.Remoteproc}},
		}
		hardware := map[health.HardwareCapability]struct{}{health.Remoteproc: {}}

		got := health.FilterByHardware(deps, hardware)

		assert.Equal(t, deps, got)
	})

	t.Run("excludes dependencies when hardware is absent", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "remoteproc-runtime", Label: "Runtime", HardwarePrerequisite: []health.HardwareCapability{health.Remoteproc}},
		}
		hardware := map[health.HardwareCapability]struct{}{}

		got := health.FilterByHardware(deps, hardware)

		assert.Empty(t, got)
	})

	t.Run("filters mixed dependencies correctly", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "spaghetti", Label: "Food"},
			{Binary: "remoteproc-runtime", Label: "Runtime", HardwarePrerequisite: []health.HardwareCapability{health.Remoteproc}},
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
