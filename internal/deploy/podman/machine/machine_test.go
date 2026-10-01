package machine_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/podman/machine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindMachineForConnection(t *testing.T) {
	for _, test := range []struct {
		name      string
		machines  []machine.Identity
		want      string
		errorText string
	}{
		{"matches both fields", []machine.Identity{{"port-only", 2222, "other"}, {"identity-only", 3333, "key"}, {"actual", 2222, "key"}}, "actual", ""},
		{"matches regardless of order", []machine.Identity{{"actual", 2222, "key"}, {"other", 3333, "key"}}, "actual", ""},
		{"ignores a matching name with a different identity", []machine.Identity{{"selected", 3333, "other"}, {"actual", 2222, "key"}}, "actual", ""},
		{"no machines", nil, "", "no Podman machine"},
		{"port alone is insufficient", []machine.Identity{{"same-name", 2222, "other"}}, "", "no Podman machine"},
		{"identity alone is insufficient", []machine.Identity{{"same-name", 3333, "key"}}, "", "no Podman machine"},
	} {
		t.Run(test.name, func(t *testing.T) {
			connection := machine.Connection{Name: "selected", URI: "ssh://core@127.0.0.1:2222/run/user/1000/podman/podman.sock", IdentityPath: "key"}

			got, err := machine.FindForConnection(connection, test.machines)

			if test.errorText != "" {
				require.ErrorContains(t, err, test.errorText)
				assert.ErrorContains(t, err, `connection "selected"`)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.want, got)
		})
	}
}

func TestFindMachineForConnectionURI(t *testing.T) {
	for _, test := range []struct {
		name     string
		uri      string
		identity string
		port     int
	}{
		{"matches root connection", "ssh://root@127.0.0.1:2222/run/podman/podman.sock", "/home/user/.local/share/containers/podman/machine/machine", 2222},
		{"matches rootless connection", "ssh://core@127.0.0.1:3333/run/user/1000/podman/podman.sock", "/home/user/.local/share/containers/podman/machine/machine", 3333},
		{"matches connection with Windows identity path", "ssh://root@127.0.0.1:4444/run/podman/podman.sock", `C:\Users\user\.local\share\containers\podman\machine\machine`, 4444},
	} {
		t.Run(test.name, func(t *testing.T) {
			connection := machine.Connection{Name: "selected", URI: test.uri, IdentityPath: test.identity}
			machines := []machine.Identity{{Name: "actual", SSHPort: test.port, IdentityPath: test.identity}}

			got, err := machine.FindForConnection(connection, machines)

			require.NoError(t, err)
			assert.Equal(t, "actual", got)
		})
	}

	t.Run("rejects malformed URI with connection context", func(t *testing.T) {
		connection := machine.Connection{Name: "selected", URI: "ssh://root@127.0.0.1:2222/%zz", IdentityPath: "key"}
		machines := []machine.Identity{{Name: "actual", SSHPort: 2222, IdentityPath: "key"}}

		got, err := machine.FindForConnection(connection, machines)

		require.ErrorContains(t, err, `connection "selected" has an invalid SSH URI`)
		assert.Empty(t, got)
	})
}
