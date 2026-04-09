package collections_test

import (
	"testing"

	"github.com/arm/topo/internal/collections"
	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	t.Run("NewSet", func(t *testing.T) {
		t.Run("creates a set containing the provided items", func(t *testing.T) {
			set := collections.NewSet("a", "b", "c")

			assert.True(t, set.Contains("a"))
			assert.True(t, set.Contains("b"))
			assert.True(t, set.Contains("c"))
			assert.Len(t, set, 3)
		})

		t.Run("deduplicates items", func(t *testing.T) {
			set := collections.NewSet("a", "a", "b")

			got := set.ToSlice()

			assert.Len(t, got, 2)
		})
	})

	t.Run("Add", func(t *testing.T) {
		t.Run("adds an item to the set", func(t *testing.T) {
			set := collections.NewSet[string]()

			set.Add("x")

			assert.True(t, set.Contains("x"))
		})
	})

	t.Run("Contains", func(t *testing.T) {
		t.Run("returns true for an item in the set", func(t *testing.T) {
			set := collections.NewSet("present")

			got := set.Contains("present")

			assert.True(t, got)
		})

		t.Run("returns false for an item not in the set", func(t *testing.T) {
			set := collections.NewSet("present")

			got := set.Contains("absent")

			assert.False(t, got)
		})
	})

	t.Run("ToSlice", func(t *testing.T) {
		t.Run("returns all elements as a slice", func(t *testing.T) {
			set := collections.NewSet(1, 2, 3)

			got := set.ToSlice()

			assert.ElementsMatch(t, []int{1, 2, 3}, got)
		})

		t.Run("returns an empty slice for an empty set", func(t *testing.T) {
			set := collections.NewSet[int]()

			got := set.ToSlice()

			assert.Empty(t, got)
		})
	})
}
