package collections_test

import (
	"testing"

	"github.com/arm/topo/internal/collections"
	"github.com/stretchr/testify/assert"
)

func TestGroupBy(t *testing.T) {
	t.Run("groups items by key returned from key function", func(t *testing.T) {
		type food struct {
			name  string
			tasty bool
		}
		tastyPizza := food{name: "pizza", tasty: true}
		tastyBurrito := food{name: "burrito", tasty: true}
		disgustingVeggies := food{name: "veggies", tasty: false}

		got := collections.GroupBy(
			[]food{tastyPizza, tastyBurrito, disgustingVeggies},
			func(f food) bool { return f.tasty },
		)

		want := []collections.Group[food, bool]{
			{Key: true, Members: []food{tastyPizza, tastyBurrito}},
			{Key: false, Members: []food{disgustingVeggies}},
		}
		assert.Equal(t, want, got)
	})

	t.Run("retains order of grouped items", func(t *testing.T) {
		type vegetable struct {
			name string
			kind string
		}
		crunchyCarrot := vegetable{kind: "crunchy", name: "carrot"}
		crunchyBroccoli := vegetable{kind: "crunchy", name: "broccoli"}
		leafySpinach := vegetable{kind: "leafy", name: "spinach"}
		leafyKale := vegetable{kind: "leafy", name: "kale"}
		softTomato := vegetable{kind: "soft", name: "tomato"}

		got := collections.GroupBy(
			[]vegetable{leafyKale, softTomato, crunchyCarrot, crunchyBroccoli, leafySpinach},
			func(v vegetable) string { return v.kind },
		)

		want := []collections.Group[vegetable, string]{
			{Key: "leafy", Members: []vegetable{leafyKale, leafySpinach}},
			{Key: "soft", Members: []vegetable{softTomato}},
			{Key: "crunchy", Members: []vegetable{crunchyCarrot, crunchyBroccoli}},
		}
		assert.Equal(t, want, got)
	})
}
