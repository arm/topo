package health_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestBinaryRegex(t *testing.T) {
	t.Run("binary regex fails an incorrect binary name", func(t *testing.T) {
		got := "bin ary"

		assert.False(t, ssh.BinaryRegex.MatchString(got))
	})

	t.Run("binary regex passes a correct binary name", func(t *testing.T) {
		got := "binary"

		assert.True(t, ssh.BinaryRegex.MatchString(got))
	})
}

func TestDependencyFormat(t *testing.T) {
	t.Run("host dependencies are of the correct format", func(t *testing.T) {
		for _, dep := range health.HostRequiredDependencies {
			assert.True(t, ssh.BinaryRegex.MatchString(dep.Binary))
		}
	})

	t.Run("target dependencies are of the correct format", func(t *testing.T) {
		for _, dep := range health.TargetRequiredDependencies {
			assert.True(t, ssh.BinaryRegex.MatchString(dep.Binary))
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
		deps := []health.Dependency{
			{Binary: "foo", Label: "bar", Checks: []health.Check{health.BinaryExists()}},
			{Binary: "baz", Label: "qux", Checks: []health.Check{health.BinaryExists()}},
		}
		mockBinaryExists := func(bin string) error {
			return fmt.Errorf("%q executable file not found in $PATH", bin)
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Binary: "foo", Label: "bar", Checks: []health.Check{health.BinaryExists()}},
				Error:      mockBinaryExists("foo"),
			},
			{
				Dependency: health.Dependency{Binary: "baz", Label: "qux", Checks: []health.Check{health.BinaryExists()}},
				Error:      mockBinaryExists("baz"),
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("wraps error as WarningError when binary exists check severity is warning", func(t *testing.T) {
		missingBin := health.Dependency{Binary: "missing-bin", Label: "Missing", Checks: []health.Check{health.BinaryExistsWarning()}}
		deps := []health.Dependency{missingBin}
		mockBinaryExists := func(bin string) error {
			return fmt.Errorf("%q executable file not found in $PATH", bin)
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		assert.Len(t, got, 1)
		want := []health.DependencyStatus{
			{
				Dependency: missingBin,
				Error:      health.WarningError{Err: mockBinaryExists("missing-bin")},
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("when a dependency is found, its status entry reflects that", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "baz", Label: "qux", Checks: []health.Check{health.BinaryExists()}},
		}
		mockBinaryExists := func(bin string) error {
			if bin == "baz" {
				return nil
			}
			return fmt.Errorf("%q executable file not found in $PATH", bin)
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Binary: "baz", Label: "qux", Checks: []health.Check{health.BinaryExists()}},
				Error:      nil,
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("omits dependency when none of its SoftwarePrerequisites are installed", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "docker", Label: "Container Engine", Checks: []health.Check{health.BinaryExists()}},
			{Binary: "runtime", Label: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists()}},
		}
		mockBinaryExists := func(bin string) error {
			if bin == "runtime" {
				return nil
			}
			return fmt.Errorf("%q executable file not found in $PATH", bin)
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Binary: "docker", Label: "Container Engine", Checks: []health.Check{health.BinaryExists()}}, Error: mockBinaryExists("docker")},
		}
		assert.Equal(t, want, got)
	})

	t.Run("checks dependency when one of its SoftwarePrerequisites is installed", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "docker", Label: "Container Engine", SoftwareEnumID: health.Docker, Checks: []health.Check{health.BinaryExists()}},
			{Binary: "runtime", Label: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists()}},
		}
		mockBinaryExists := func(bin string) error {
			return nil
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Binary: "docker", Label: "Container Engine", SoftwareEnumID: health.Docker, Checks: []health.Check{health.BinaryExists()}}, Error: nil},
			{Dependency: health.Dependency{Binary: "runtime", Label: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists()}}, Error: nil},
		}
		assert.Equal(t, want, got)
	})

	t.Run("captures Fix from failing check", func(t *testing.T) {
		dep := health.Dependency{
			Binary: "vader.exe",
			Label:  "Sith",
			Checks: []health.Check{{Kind: health.CheckBinaryExists, Severity: health.SeverityWarning, Fix: "turn Anakin into a bad man"}},
		}
		mockBinaryExists := func(bin string) error {
			return errors.New("vader not found")
		}

		got := health.PerformChecks([]health.Dependency{dep}, mockBinaryExists)

		assert.Len(t, got, 1)
		assert.Equal(t, "turn Anakin into a bad man", got[0].Fix)
	})

	t.Run("checks dependency with no SoftwarePrerequisites unconditionally", func(t *testing.T) {
		deps := []health.Dependency{
			{Binary: "standalone", Label: "Tools", Checks: []health.Check{health.BinaryExists()}},
		}
		mockBinaryExists := func(bin string) error {
			return nil
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Binary: "standalone", Label: "Tools", Checks: []health.Check{health.BinaryExists()}}, Error: nil},
		}
		assert.Equal(t, want, got)
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
