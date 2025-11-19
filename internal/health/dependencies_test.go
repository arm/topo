package health_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestBinaryRegex(t *testing.T) {
	t.Run("binary regex passes a correct binary name", func(t *testing.T) {
		got := "bin ary"

		assert.False(t, health.BinaryRegex.MatchString(got))
	})

	t.Run("binary regex fails an incorrect binary name", func(t *testing.T) {
		got := "binary"

		assert.True(t, health.BinaryRegex.MatchString(got))
	})
}

func TestDependencyFormat(t *testing.T) {
	t.Run("host dependencies are of the correct format", func(t *testing.T) {
		for _, dep := range health.HostRequiredDependencies {
			assert.True(t, health.BinaryRegex.MatchString(dep.Name))
		}
	})

	t.Run("target dependencies are of the correct format", func(t *testing.T) {
		for _, dep := range health.TargetRequiredDependencies {
			assert.True(t, health.BinaryRegex.MatchString(dep.Name))
		}
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
