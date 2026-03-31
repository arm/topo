package pubkeytransfer_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRunner struct {
	mock.Mock
}

func (m *mockRunner) RunWithStdin(cmd string, stdin []byte) (string, error) {
	args := m.Called(cmd, stdin)
	return args.String(0), args.Error(1)
}

func TestPubKeyTransfer(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		t.Run("transfers the public key to the target", func(t *testing.T) {
			tmp := t.TempDir()
			privKeyPath := filepath.Join(tmp, "id_ed25519_testrun")
			pubKeyContent := []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItestkey")
			require.NoError(t, os.WriteFile(privKeyPath+".pub", pubKeyContent, 0o600))

			runner := &mockRunner{}
			runner.On(
				"RunWithStdin",
				mock.MatchedBy(func(cmd string) bool {
					return strings.Contains(cmd, "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys")
				}),
				pubKeyContent,
			).Return("ssh invoked", nil)

			op := pubkeytransfer.NewPubKeyTransfer(privKeyPath, runner)

			var buf bytes.Buffer
			require.NoError(t, op.Run(&buf))
			require.Contains(t, buf.String(), "ssh invoked")
			runner.AssertExpectations(t)
		})
	})
}
