package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectsInSourceOrder(t *testing.T) {
	t.Run("returns projects ordered to match sources", func(t *testing.T) {
		sources := []GitHubSource{
			{Repo: "example/first", SHA: "first-sha"},
			{Repo: "example/second", SHA: "second-sha"},
			{Repo: "example/third", SHA: "third-sha"},
		}
		projects := []Project{
			{URL: "https://github.com/example/third.git", Ref: "third-sha"},
			{URL: "https://github.com/example/orphan.git", Ref: "orphan-sha"},
			{URL: "https://github.com/example/second.git", Ref: "second-sha"},
			{URL: "https://github.com/example/first.git", Ref: "first-sha"},
		}

		got := ProjectsInSourceOrder(sources, projects)

		want := []Project{
			{URL: "https://github.com/example/first.git", Ref: "first-sha"},
			{URL: "https://github.com/example/second.git", Ref: "second-sha"},
			{URL: "https://github.com/example/third.git", Ref: "third-sha"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("panics when source has no matching project", func(t *testing.T) {
		sources := []GitHubSource{
			{Repo: "example/missing", SHA: "missing-sha"},
		}
		projects := []Project{}

		assert.PanicsWithValue(t, "missing project for source example/missing@missing-sha", func() {
			ProjectsInSourceOrder(sources, projects)
		})
	})
}
