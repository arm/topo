package dependencies

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
