package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveConfiguredUser(t *testing.T) {
	t.Run("IP literal with no explicit config returns dest user", func(t *testing.T) {
		config := []byte(`debug1: /etc/ssh/ssh_config line 57: Applying options for *
user username
hostname 10.2.2.26
`)
		dest := Destination{User: "root", Host: "10.2.2.26"}

		got, err := resolveConfiguredUser(dest, config)

		assert.NoError(t, err)
		assert.Equal(t, "root", got)
	})

	t.Run("IP literal with no explicit config and no dest user returns config user", func(t *testing.T) {
		config := []byte(`debug1: /etc/ssh/ssh_config line 57: Applying options for *
user username
hostname 10.2.2.26
`)
		dest := Destination{Host: "10.2.2.26"}

		got, err := resolveConfiguredUser(dest, config)

		assert.NoError(t, err)
		assert.Equal(t, "username", got)
	})

	t.Run("non-IP alias with no explicit config returns error", func(t *testing.T) {
		config := []byte(`debug1: /etc/ssh/ssh_config line 57: Applying options for *
user username
hostname board-alias
`)
		dest := Destination{User: "root", Host: "board-alias"}

		_, err := resolveConfiguredUser(dest, config)

		assert.ErrorContains(t, err, "no explicit host config found for board-alias")
	})

	t.Run("explicit config with no user conflict returns config user", func(t *testing.T) {
		config := []byte(`debug1: /tmp/.ssh/topo_config line 1: Applying options for board-alias
user root
hostname 10.2.2.26
debug1: /etc/ssh/ssh_config line 57: Applying options for *
`)
		dest := Destination{User: "root", Host: "board-alias"}

		got, err := resolveConfiguredUser(dest, config)

		assert.NoError(t, err)
		assert.Equal(t, "root", got)
	})

	t.Run("explicit config with different user returns error", func(t *testing.T) {
		config := []byte(`debug1: /tmp/.ssh/topo_config line 1: Applying options for board-alias
user root
hostname 10.2.2.26
debug1: /etc/ssh/ssh_config line 57: Applying options for *
`)
		dest := Destination{User: "admin", Host: "board-alias"}

		_, err := resolveConfiguredUser(dest, config)

		assert.ErrorContains(t, err, `ssh host/alias "board-alias" is already associated with user "root"`)
	})

	t.Run("explicit config with no dest user returns config user", func(t *testing.T) {
		config := []byte(`debug1: /tmp/.ssh/topo_config line 1: Applying options for board-alias
user root
hostname 10.2.2.26
debug1: /etc/ssh/ssh_config line 57: Applying options for *
`)
		dest := Destination{Host: "board-alias"}

		got, err := resolveConfiguredUser(dest, config)

		assert.NoError(t, err)
		assert.Equal(t, "root", got)
	})

	t.Run("IP literal with explicit config and different user returns error", func(t *testing.T) {
		config := []byte(`debug1: /tmp/.ssh/topo_config line 1: Applying options for 10.2.2.26
user root
hostname 10.2.2.26
debug1: /etc/ssh/ssh_config line 57: Applying options for *
`)
		dest := Destination{User: "admin", Host: "10.2.2.26"}

		_, err := resolveConfiguredUser(dest, config)

		assert.ErrorContains(t, err, `ssh host/alias "10.2.2.26" is already associated with user "root"`)
	})
}
