package sshkeygen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys/sshkeygen"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestSSHKeyGenRun(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "keys", "id_ed25519_test")
	opts := sshkeygen.SSHKeyGenOptions{
		WithMockKeyGen: func(keyType, keyPath, targetHost string) *exec.Cmd {
			return testutil.CmdWithOutput("ssh-keygen invoked", 0)
		},
	}
	op := sshkeygen.NewSSHKeyGen("Generate SSH key pair", ssh.NewDestination("user@example.com"), "ed25519", keyPath, opts)
	var buf bytes.Buffer
	require.NoError(t, op.Run(&buf))
	require.Contains(t, buf.String(), "ssh-keygen invoked")
	_, err := os.Stat(filepath.Dir(keyPath))
	require.NoError(t, err, "expected key directory to be created")
}
