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
