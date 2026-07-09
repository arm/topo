package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlanUpdate(t *testing.T) {
	t.Run("classifies added, updated and removed projects", func(t *testing.T) {
		sources := []GitHubSource{
			{Repo: "example/unchanged", SHA: "same-sha"},
			{Repo: "example/updated", SHA: "new-sha"},
			{Repo: "example/added", SHA: "added-sha"},
		}
		current := []Project{
			{URL: "https://github.com/example/removed.git", Ref: "removed-sha"},
			{URL: "https://github.com/example/unchanged.git", Ref: "same-sha"},
			{URL: "https://github.com/example/updated.git", Ref: "old-sha"},
		}

		got := PlanUpdate(sources, current)

		want := UpdatePlan{
			ToAdd:     []GitHubSource{{Repo: "example/added", SHA: "added-sha"}},
			ToUpdate:  []GitHubSource{{Repo: "example/updated", SHA: "new-sha"}},
			ToRemove:  []Project{{URL: "https://github.com/example/removed.git", Ref: "removed-sha"}},
			Unchanged: []Project{{URL: "https://github.com/example/unchanged.git", Ref: "same-sha"}},
		}
		assert.Equal(t, want, got)
	})
}

func TestUpdatePlan(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		t.Run("returns false when only projects are unchanged", func(t *testing.T) {
			plan := UpdatePlan{
				Unchanged: []Project{{URL: "https://github.com/example/unchanged.git", Ref: "same-sha"}},
			}

			got := plan.HasChanges()

			assert.False(t, got)
		})

		t.Run("returns true when projects will be added", func(t *testing.T) {
			plan := UpdatePlan{
				ToAdd: []GitHubSource{{Repo: "example/added", SHA: "added-sha"}},
			}

			got := plan.HasChanges()

			assert.True(t, got)
		})

		t.Run("returns true when projects will be updated", func(t *testing.T) {
			plan := UpdatePlan{
				ToUpdate: []GitHubSource{{Repo: "example/updated", SHA: "new-sha"}},
			}

			got := plan.HasChanges()

			assert.True(t, got)
		})

		t.Run("returns true when projects will be removed", func(t *testing.T) {
			plan := UpdatePlan{
				ToRemove: []Project{{URL: "https://github.com/example/removed.git", Ref: "removed-sha"}},
			}

			got := plan.HasChanges()

			assert.True(t, got)
		})
	})
}
