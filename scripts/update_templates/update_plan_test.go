package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlanUpdate(t *testing.T) {
	t.Run("classifies added, updated and removed templates", func(t *testing.T) {
		sources := []GitHubSource{
			{Repo: "example/unchanged", SHA: "same-sha"},
			{Repo: "example/updated", SHA: "new-sha"},
			{Repo: "example/added", SHA: "added-sha"},
		}
		current := []Template{
			{URL: "https://github.com/example/removed.git", Ref: "removed-sha"},
			{URL: "https://github.com/example/unchanged.git", Ref: "same-sha"},
			{URL: "https://github.com/example/updated.git", Ref: "old-sha"},
		}

		got := PlanUpdate(sources, current)

		want := UpdatePlan{
			ToAdd:     []GitHubSource{{Repo: "example/added", SHA: "added-sha"}},
			ToUpdate:  []GitHubSource{{Repo: "example/updated", SHA: "new-sha"}},
			ToRemove:  []Template{{URL: "https://github.com/example/removed.git", Ref: "removed-sha"}},
			Unchanged: []Template{{URL: "https://github.com/example/unchanged.git", Ref: "same-sha"}},
		}
		assert.Equal(t, want, got)
	})
}

func TestUpdatePlan(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		t.Run("returns false when only templates are unchanged", func(t *testing.T) {
			plan := UpdatePlan{
				Unchanged: []Template{{URL: "https://github.com/example/unchanged.git", Ref: "same-sha"}},
			}

			got := plan.HasChanges()

			assert.False(t, got)
		})

		t.Run("returns true when templates will be added", func(t *testing.T) {
			plan := UpdatePlan{
				ToAdd: []GitHubSource{{Repo: "example/added", SHA: "added-sha"}},
			}

			got := plan.HasChanges()

			assert.True(t, got)
		})

		t.Run("returns true when templates will be updated", func(t *testing.T) {
			plan := UpdatePlan{
				ToUpdate: []GitHubSource{{Repo: "example/updated", SHA: "new-sha"}},
			}

			got := plan.HasChanges()

			assert.True(t, got)
		})

		t.Run("returns true when templates will be removed", func(t *testing.T) {
			plan := UpdatePlan{
				ToRemove: []Template{{URL: "https://github.com/example/removed.git", Ref: "removed-sha"}},
			}

			got := plan.HasChanges()

			assert.True(t, got)
		})
	})
}
