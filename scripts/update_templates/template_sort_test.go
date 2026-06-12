package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplatesInSourceOrder(t *testing.T) {
	t.Run("returns templates ordered to match sources", func(t *testing.T) {
		sources := []GitHubSource{
			{Repo: "example/first", SHA: "first-sha"},
			{Repo: "example/second", SHA: "second-sha"},
			{Repo: "example/third", SHA: "third-sha"},
		}
		templates := []Template{
			{URL: "https://github.com/example/third.git", Ref: "third-sha"},
			{URL: "https://github.com/example/orphan.git", Ref: "orphan-sha"},
			{URL: "https://github.com/example/second.git", Ref: "second-sha"},
			{URL: "https://github.com/example/first.git", Ref: "first-sha"},
		}

		got := TemplatesInSourceOrder(sources, templates)

		want := []Template{
			{URL: "https://github.com/example/first.git", Ref: "first-sha"},
			{URL: "https://github.com/example/second.git", Ref: "second-sha"},
			{URL: "https://github.com/example/third.git", Ref: "third-sha"},
		}
		assert.Equal(t, want, got)
	})
}
