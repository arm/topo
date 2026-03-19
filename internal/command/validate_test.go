package command_test

import (
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
)

func TestValidateBinaryName(t *testing.T) {
	t.Run("returns error for invalid binary name", func(t *testing.T) {
		err := command.ValidateBinaryName("bin ary")

		assert.Error(t, err)
	})

	t.Run("returns no error for valid binary name", func(t *testing.T) {
		err := command.ValidateBinaryName("binary")

		assert.NoError(t, err)
	})
}
