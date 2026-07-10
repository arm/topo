package operations

import (
	"context"
	"fmt"
	"io"
	"os"
)

const remoteAuthorizedKeysCommand = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

var passwordAuthArgs = []string{
	"-o", "PreferredAuthentications=password",
}

type sshRunnerWithExtraArgs interface {
	RunWithStdinAndArgs(ctx context.Context, command string, stdin []byte, sshArgs ...string) (string, error)
}

type PubKeyTransfer struct {
	pubKeyPath string
	r          sshRunnerWithExtraArgs
}

func NewPubKeyTransfer(privKeyPath string, r sshRunnerWithExtraArgs) *PubKeyTransfer {
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

	cmdOutput, err := kt.r.RunWithStdinAndArgs(context.TODO(), remoteAuthorizedKeysCommand, pubKey, passwordAuthArgs...)
	if err != nil {
		return fmt.Errorf("failed to transfer public key to target: %w", err)
	}
	_, err = outputWriter.Write([]byte(cmdOutput))
	return err
}
