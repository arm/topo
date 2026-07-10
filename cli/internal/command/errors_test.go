package command_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
)

func TestFormatError(t *testing.T) {
	t.Run("it formats the error message with the command and original error", func(t *testing.T) {
		args := []string{"ssh", "-L", "8080:localhost:80", "user@remote"}
		originalErr := fmt.Errorf("connection refused")

		err := command.FormatError(args, originalErr)

		assert.Error(t, err)
		assert.Equal(t, err.Error(), "ssh -L 8080:localhost:80 user@remote failed: connection refused")
	})
}
