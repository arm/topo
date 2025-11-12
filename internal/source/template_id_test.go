package source_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/stretchr/testify/assert"
)

func TestTemplateId(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		t.Run("returns template ID in correct format", func(t *testing.T) {
			src := source.TemplateId("hello-world")
			assert.Equal(t, "template:hello-world", src.String())
		})
	})
}
