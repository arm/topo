package operations

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/arm/topo/internal/runner"
)

const remoteAuthorizedKeysCommand = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

var passwordAuthArgs = []string{
	"-o", "PreferredAuthentications=password",
}

type PubKeyTransfer struct {
	pubKeyPath string
	r          *runner.SSH
}

func NewPubKeyTransfer(privKeyPath string, r *runner.SSH) *PubKeyTransfer {
	return &PubKeyTransfer{
		pubKeyPath: privKeyPath + ".pub",
		r:          r,
	}
}

func (kt *PubKeyTransfer) Description() string {
	return "Transfer public key to target and set it as an authorized key"
}

func (kt *PubKeyTransfer) Run(outputWriter io.Writer) error {
	pubKey, err := os.ReadFile(kt.pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key %s: %w", kt.pubKeyPath, err)
	}

	cmdStdout, _, err := kt.r.RunShellWithStdinAndSSHArgs(context.TODO(), remoteAuthorizedKeysCommand, pubKey, passwordAuthArgs...)
	if err != nil {
		return fmt.Errorf("failed to transfer public key to target: %w", err)
	}
	_, err = outputWriter.Write([]byte(cmdStdout))
	return err
}
