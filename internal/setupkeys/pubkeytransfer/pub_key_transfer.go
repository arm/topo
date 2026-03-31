package pubkeytransfer

import (
	"fmt"
	"io"
	"os"

	"github.com/arm/topo/internal/command"
)

const remoteAuthorizedKeysCommand = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

type runner interface {
	RunWithStdin(command string, stdin []byte) (string, error)
}

type PubKeyTransfer struct {
	pubKeyPath string
	runner     runner
}

func NewPubKeyTransfer(privKeyPath string, runner runner) *PubKeyTransfer {
	return &PubKeyTransfer{
		pubKeyPath: privKeyPath + ".pub",
		runner:     runner,
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

	cmdOutput, err := kt.runner.RunWithStdin(command.WrapInLoginShell(remoteAuthorizedKeysCommand), pubKey)
	if err != nil {
		return fmt.Errorf("failed to transfer public key to target: %w", err)
	}
	_, err = outputWriter.Write([]byte(cmdOutput))
	return err
}
