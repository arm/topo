package pubkeytransfer_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestPubKeyTransferDryRun(t *testing.T) {
	tmp := t.TempDir()
	privKeyPath := filepath.Join(tmp, "id_ed25519_testrun")
	op := pubkeytransfer.NewPubKeyTransfer("Transfer public key", ssh.NewDestination("hola@chau"), privKeyPath, pubkeytransfer.PubKeyTransferOptions{})

	var buf bytes.Buffer
	require.NoError(t, op.DryRun(&buf))
	output := buf.String()
	require.Contains(t, output, "ssh -- ssh://hola@chau")
	require.Contains(t, output, "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys")
}

func TestPubKeyTransferRun(t *testing.T) {
	tmp := t.TempDir()
	privKeyPath := filepath.Join(tmp, "id_ed25519_testdryrun")
	pubKeyPath := privKeyPath + ".pub"
	pubKeyContent := []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItestkey")
	require.NoError(t, os.WriteFile(pubKeyPath, pubKeyContent, 0o600))

	type call struct {
		dest  ssh.Destination
		cmd   string
		stdin []byte
		args  []string
	}
	var got call

	opts := pubkeytransfer.PubKeyTransferOptions{WithMockExec: func(d ssh.Destination, command string, stdin []byte, args ...string) *exec.Cmd {
		got = call{dest: d, cmd: command, stdin: stdin, args: args}
		cmd := testutil.CmdWithOutput("ssh invoked", 0)
		if stdin != nil {
			cmd.Stdin = bytes.NewReader(stdin)
		}
		return cmd
	}}
	op := pubkeytransfer.NewPubKeyTransfer("Transfer public key", ssh.NewDestination("thing1@thing2.com"), privKeyPath, opts)

	var buf bytes.Buffer
	require.NoError(t, op.Run(&buf))
	require.Contains(t, buf.String(), "ssh invoked")
	require.Equal(t, ssh.NewDestination("thing1@thing2.com"), got.dest)
	require.Contains(t, got.cmd, "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys")
	require.Equal(t, pubKeyContent, got.stdin)
	require.Empty(t, got.args)
}
