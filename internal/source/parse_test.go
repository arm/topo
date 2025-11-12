package source_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("parses template source", func(t *testing.T) {
		gotType, gotValue, err := source.Parse("template:hello")

		require.NoError(t, err)
		assert.Equal(t, "template", gotType)
		assert.Equal(t, "hello", gotValue)
	})

	t.Run("parses git HTTPS source", func(t *testing.T) {
		gotType, gotValue, err := source.Parse("git:https://github.com/user/repo.git")

		require.NoError(t, err)
		assert.Equal(t, "git", gotType)
		assert.Equal(t, "https://github.com/user/repo.git", gotValue)
	})

	t.Run("parses git SSH source", func(t *testing.T) {
		gotType, gotValue, err := source.Parse("git:git@github.com:user/repo.git")

		require.NoError(t, err)
		assert.Equal(t, "git", gotType)
		assert.Equal(t, "git@github.com:user/repo.git", gotValue)
	})

	t.Run("preserves multiple colons in URL", func(t *testing.T) {
		gotType, gotValue, err := source.Parse("git:https://example.com:8080/repo.git")

		require.NoError(t, err)
		assert.Equal(t, "git", gotType)
		assert.Equal(t, "https://example.com:8080/repo.git", gotValue)
	})

	t.Run("returns error when colon is missing", func(t *testing.T) {
		_, _, err := source.Parse("template-ubuntu")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source format")
	})

	t.Run("returns error when value is empty", func(t *testing.T) {
		_, _, err := source.Parse("template:")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "source value cannot be empty")
	})

	t.Run("returns error when source is empty", func(t *testing.T) {
		_, _, err := source.Parse("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source format")
	})
}
