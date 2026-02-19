package health_test

import (
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestBinaryRegex(t *testing.T) {
	t.Run("binary regex passes a correct binary name", func(t *testing.T) {
		got := "bin ary"

		assert.False(t, ssh.BinaryRegex.MatchString(got))
	})

	t.Run("binary regex fails an incorrect binary name", func(t *testing.T) {
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

func TestCheckInstalled(t *testing.T) {
	mockDependencies := []health.Dependency{
		{Name: "foo", Category: "bar"},
		{Name: "baz", Category: "qux"},
	}

	t.Run("when no dependencies are found, statuses show not installed", func(t *testing.T) {
		mockBinaryExists := func(bin string) (bool, error) {
			return false, nil
		}

		got := health.CheckInstalled(mockDependencies, mockBinaryExists)

		want := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "foo", Category: "bar"},
				Installed:  false,
			},
			{
				Dependency: health.Dependency{Name: "baz", Category: "qux"},
				Installed:  false,
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("when a dependency is found, its status entry reflects that", func(t *testing.T) {
		mockBinaryExists := func(bin string) (bool, error) {
			return bin == "baz", nil
		}

		got := health.CheckInstalled(mockDependencies, mockBinaryExists)

		want := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "foo", Category: "bar"},
				Installed:  false,
			},
			{
				Dependency: health.Dependency{Name: "baz", Category: "qux"},
				Installed:  true,
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("omits dependency when none of its SoftwarePrerequisites are installed", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "docker", Category: "Container Engine"},
			{Name: "runtime", Category: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker}},
		}
		mockBinaryExists := func(bin string) (bool, error) {
			return bin == "runtime", nil
		}

		got := health.CheckInstalled(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "docker", Category: "Container Engine"}, Installed: false},
		}
		assert.Equal(t, want, got)
	})

	t.Run("checks dependency when one of its SoftwarePrerequisites is installed", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "docker", Category: "Container Engine", SoftwareEnumID: health.Docker},
			{Name: "podman", Category: "Container Engine", SoftwareEnumID: health.Podman},
			{Name: "runtime", Category: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker, health.Podman}},
		}
		mockBinaryExists := func(bin string) (bool, error) {
			return bin == "podman" || bin == "runtime", nil
		}

		got := health.CheckInstalled(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "docker", Category: "Container Engine", SoftwareEnumID: health.Docker}, Installed: false},
			{Dependency: health.Dependency{Name: "podman", Category: "Container Engine", SoftwareEnumID: health.Podman}, Installed: true},
			{Dependency: health.Dependency{Name: "runtime", Category: "Runtime", SoftwarePrerequisites: []health.SoftwareDependency{health.Docker, health.Podman}}, Installed: true},
		}
		assert.Equal(t, want, got)
	})

	t.Run("checks dependency with no SoftwarePrerequisites unconditionally", func(t *testing.T) {
		deps := []health.Dependency{
			{Name: "standalone", Category: "Tools"},
		}
		mockBinaryExists := func(bin string) (bool, error) {
			return true, nil
		}

		got := health.CheckInstalled(deps, mockBinaryExists)

		want := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "standalone", Category: "Tools"}, Installed: true},
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
			{Name: "docker", Category: "Container Engine"},
			{Name: "remoteproc-runtime", Category: "Runtime", HardwarePrerequisite: []health.HardwareCapability{health.Remoteproc}},
			{Name: "podman", Category: "Container Engine"},
		}
		hardware := map[health.HardwareCapability]struct{}{}

		got := health.FilterByHardware(deps, hardware)

		want := []health.Dependency{
			{Name: "docker", Category: "Container Engine"},
			{Name: "podman", Category: "Container Engine"},
		}
		assert.Equal(t, want, got)
	})
}

func TestCollectAvailableByCategory(t *testing.T) {
	t.Run("when a tool is installed, it is included in its category", func(t *testing.T) {
		installedDependency := health.Dependency{Name: "foo", Category: "bar"}
		dependencyStatuses := []health.DependencyStatus{
			{
				Dependency: installedDependency,
				Installed:  true,
			},
			{
				Dependency: health.Dependency{Name: "baz", Category: "bar"},
				Installed:  false,
			},
		}

		got := health.CollectAvailableByCategory(dependencyStatuses)

		want := []health.DependencyStatus{
			{
				Dependency: installedDependency,
				Installed:  true,
			},
		}
		assert.Equal(t, want, got["bar"])
	})

	t.Run("when no tools in given category are installed, category is empty", func(t *testing.T) {
		dependencyStatuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "foo", Category: "bar"},
				Installed:  false,
			},
			{
				Dependency: health.Dependency{Name: "baz", Category: "bar"},
				Installed:  false,
			},
		}

		got := health.CollectAvailableByCategory(dependencyStatuses)

		satisfyingDependencies, ok := got["bar"]
		assert.True(t, ok)
		assert.Empty(t, satisfyingDependencies)
	})
}
