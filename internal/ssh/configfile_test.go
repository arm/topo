package ssh_test

import (
	"testing"

	"github.com/arm/topo/internal/testutil"
)

func TestCreateOrModifyConfigFile(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetHomeDir(t, tmp)

	t.Run("writes include directive to config file", func(t *testing.T) {
	})

	t.Run("creates config file if it does not exist", func(t *testing.T) {
	})

	t.Run("does not duplicate include directive if it already exists", func(t *testing.T) {
	})

	t.Run("adds new entry to existing config file", func(t *testing.T) {
	})

	t.Run("modifies existing entry in config file, preserving unmodified directives", func(t *testing.T) {
	})
}
