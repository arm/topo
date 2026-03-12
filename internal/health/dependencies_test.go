package health_test

import (
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
			assert.True(t, ssh.BinaryRegex.MatchString(dep.Name))
		}
	})

	t.Run("target dependencies are of the correct format", func(t *testing.T) {
		for _, dep := range health.TargetRequiredDependencies {
			assert.True(t, ssh.BinaryRegex.MatchString(dep.Name))
		}
	})

	t.Run("target SoftwarePrerequisites reference valid dependencies", func(t *testing.T) {
		availableEnums := make(map[health.SoftwareDependency]bool)
		seenEnums := make(map[health.SoftwareDependency]string)

		t.Run("There are no duplicate SoftwareEnumID assignments", func(t *testing.T) {
			for _, dep := range health.TargetRequiredDependencies {
				if dep.SoftwareEnumID != health.UnsetSoftwareDependency {
					if existingDep, exists := seenEnums[dep.SoftwareEnumID]; exists {
						t.Errorf("Duplicate SoftwareEnumID %d assigned to both %q and %q", dep.SoftwareEnumID, existingDep, dep.Name)
					}
					seenEnums[dep.SoftwareEnumID] = dep.Name
					availableEnums[dep.SoftwareEnumID] = true
				}
			}
		})

		t.Run("all SoftwarePrerequisites reference valid SoftwareEnumID", func(t *testing.T) {
			for _, dep := range health.TargetRequiredDependencies {
				for _, prereq := range dep.SoftwarePrerequisites {
					assert.True(t, availableEnums[prereq], "%q has SoftwarePrerequisites %v which is not provided by any dependency's SoftwareEnumID", dep.Name, prereq)
				}
			}
		})
	})
}

func TestPerformChecks(t *testing.T) {
	mockDependencies := []health.Dependency{
		{Name: "foo", Category: "bar", Checks: []health.Check{health.BinaryExists("foo")}},
		{Name: "baz", Category: "qux", Checks: []health.Check{health.BinaryExists("baz")}},
	}

	t.Run("when no dependencies are found, statuses show not installed", func(t *testing.T) {
		mockBinaryExists := func(bin string) error {
			return fmt.Errorf("%q executable file not found in $PATH", bin)
		}

		got := health.PerformChecks(mockDependencies, mockBinaryExists)

		want := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "foo", Category: "bar", Checks: []health.Check{health.BinaryExists("foo")}},
				Error:      mockBinaryExists("foo"),
			},
			{
				Dependency: health.Dependency{Name: "baz", Category: "qux", Checks: []health.Check{health.BinaryExists("baz")}},
				Error:      mockBinaryExists("baz"),
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("when a dependency is found, its status entry reflects that", func(t *testing.T) {
		mockBinaryExists := func(bin string) error {
			if bin == "baz" {
				return nil
			}
			return fmt.Errorf("%q executable file not found in $PATH", bin)
		}

		got := health.PerformChecks(mockDependencies, mockBinaryExists)

		want := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "foo", Category: "bar", Checks: []health.Check{health.BinaryExists("foo")}},
				Error:      mockBinaryExists("foo"),
			},
			{
				Dependency: health.Dependency{Name: "baz", Category: "qux", Checks: []health.Check{health.BinaryExists("baz")}},
				Error:      nil,
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("omits dependency when none of its SoftwarePrerequisites are installed", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "docker", Category: "Container Engine", Checks: []health.Check{health.BinaryExists("docker")}},
			{Name: "runtime", Category: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists("runtime")}},
		}
		mockBinaryExists := func(bin string) error {
			if bin == "runtime" {
				return nil
			}
			return fmt.Errorf("%q executable file not found in $PATH", bin)
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "docker", Category: "Container Engine", Checks: []health.Check{health.BinaryExists("docker")}}, Error: mockBinaryExists("docker")},
		}
		assert.Equal(t, want, got)
	})

	t.Run("checks dependency when one of its SoftwarePrerequisites is installed", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "docker", Category: "Container Engine", SoftwareEnumID: health.Docker, Checks: []health.Check{health.BinaryExists("docker")}},
			{Name: "runtime", Category: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists("runtime")}},
		}
		mockBinaryExists := func(bin string) error {
			return nil
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "docker", Category: "Container Engine", SoftwareEnumID: health.Docker, Checks: []health.Check{health.BinaryExists("docker")}}, Error: nil},
			{Dependency: health.Dependency{Name: "runtime", Category: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}, Checks: []health.Check{health.BinaryExists("runtime")}}, Error: nil},
		}
		assert.Equal(t, want, got)
	})

	t.Run("checks dependency with no SoftwarePrerequisites unconditionally", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "standalone", Category: "Tools", Checks: []health.Check{health.BinaryExists("standalone")}},
		}
		mockBinaryExists := func(bin string) error {
			return nil
		}

		got := health.PerformChecks(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "standalone", Category: "Tools", Checks: []health.Check{health.BinaryExists("standalone")}}, Error: nil},
		}
		assert.Equal(t, want, got)
	})
}

func TestFilterByHardware(t *testing.T) {
	t.Run("includes dependencies with no hardware requirement", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "docker", Category: "Container Engine"},
		}
		hardware := map[health.HardwareCapability]struct{}{}

		got := health.FilterByHardware(deps, hardware)

		assert.Equal(t, deps, got)
	})

	t.Run("includes dependencies when hardware is present", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "remoteproc-runtime", Category: "Runtime", HardwarePrerequisite: []health.HardwareCapability{health.Remoteproc}},
		}
		hardware := map[health.HardwareCapability]struct{}{health.Remoteproc: {}}

		got := health.FilterByHardware(deps, hardware)

		assert.Equal(t, deps, got)
	})

	t.Run("excludes dependencies when hardware is absent", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "remoteproc-runtime", Category: "Runtime", HardwarePrerequisite: []health.HardwareCapability{health.Remoteproc}},
		}
		hardware := map[health.HardwareCapability]struct{}{}

		got := health.FilterByHardware(deps, hardware)

		assert.Empty(t, got)
	})

	t.Run("filters mixed dependencies correctly", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "spaghetti", Category: "Food"},
			{Name: "remoteproc-runtime", Category: "Runtime", HardwarePrerequisite: []health.HardwareCapability{health.Remoteproc}},
			{Name: "pizza", Category: "Food"},
		}

		got := health.FilterByHardware(deps, nil)

		want := []health.Dependency{
			{Name: "spaghetti", Category: "Food"},
			{Name: "pizza", Category: "Food"},
		}
		assert.Equal(t, want, got)
	})
}
