package source_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/stretchr/testify/assert"
)

func TestGit(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		t.Run("returns git:URL@ref when ref is set", func(t *testing.T) {
			src := source.Git{
				URL: "https://github.com/example/test.git",
				Ref: "v1.0",
			}
			assert.Equal(t, "git:https://github.com/example/test.git@v1.0", src.String())
		})

		t.Run("returns git:URL when ref is empty", func(t *testing.T) {
			src := source.Git{
				URL: "https://github.com/example/test.git",
				Ref: "",
			}
			assert.Equal(t, "git:https://github.com/example/test.git", src.String())
		})
	})
}
