package operations_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/testutil"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/setupkeys/operations"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPubKeyTransfer(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		t.Run("transfers the public key to the target", func(t *testing.T) {
			tmp := t.TempDir()
			privKeyPath := filepath.Join(tmp, "id_ed25519_testrun")
			pubKeyContent := []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItestkey")
			testutil.RequireWriteFile(t, privKeyPath+".pub", string(pubKeyContent))

			r := &runner.Mock{}
			r.On(
				"RunWithStdinAndArgs",
				mock.Anything,
				mock.MatchedBy(func(cmd string) bool {
					return strings.Contains(cmd, "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys")
				}),
				pubKeyContent,
				"-o",
				"PreferredAuthentications=password",
			).Return("ssh invoked", nil)

			op := operations.NewPubKeyTransfer(privKeyPath, r)

			var buf bytes.Buffer
			require.NoError(t, op.Run(&buf))
			require.Contains(t, buf.String(), "ssh invoked")
			r.AssertExpectations(t)
		})
	})
}
