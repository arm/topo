package dependencies

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	mockDependencies := []Dependency{
		{Name: "foo", Category: "bar"},
		{Name: "baz", Category: "qux"},
	}

	t.Run("when no dependencies are found, statuses show not installed", func(t *testing.T) {
		mockBinaryExists := func(bin string) bool {
			return false
		}

		got := Check(mockDependencies, mockBinaryExists)

		want := []Status{
			{Dependency{Name: "foo", Category: "bar"}, false},
			{Dependency{Name: "baz", Category: "qux"}, false},
		}
		assert.Equal(t, want, got)
	})

	t.Run("when a dependency is found, its status entry reflects that", func(t *testing.T) {
		mockBinaryExists := func(bin string) bool {
			return bin == "baz"
		}

		got := Check(mockDependencies, mockBinaryExists)

		want := []Status{
			{Dependency{Name: "foo", Category: "bar"}, false},
			{Dependency{Name: "baz", Category: "qux"}, true},
		}
		assert.Equal(t, want, got)
	})
}

func TestCollectAvailableByCategory(t *testing.T) {
	t.Run("when a tool is installed, it is included in its category", func(t *testing.T) {
		installedDependency := Dependency{Name: "foo", Category: "bar"}
		dependencyStatuses := []Status{
			{
				Dependency: installedDependency,
				Installed:  true,
			},
			{
				Dependency: Dependency{Name: "baz", Category: "bar"},
				Installed:  false,
			},
		}

		got := CollectAvailableByCategory(dependencyStatuses)

		want := []Status{
			{
				Dependency: installedDependency,
				Installed:  true,
			},
		}
		assert.Equal(t, want, got["bar"])
	})

	t.Run("when no tools in given category are installed, category is empty", func(t *testing.T) {
		dependencyStatuses := []Status{
			{
				Dependency: Dependency{Name: "foo", Category: "bar"},
				Installed:  false,
			},
			{
				Dependency: Dependency{Name: "baz", Category: "bar"},
				Installed:  false,
			},
		}

		got := CollectAvailableByCategory(dependencyStatuses)

		satisfyingDependencies, ok := got["bar"]
		assert.True(t, ok)
		assert.Empty(t, satisfyingDependencies)
	})
}
