package pubkeytransfer

import (
	"fmt"
	"io"
	"os"

	"github.com/arm/topo/internal/ssh"
	target "github.com/arm/topo/internal/target"
)

const remoteAuthorizedKeysCommand = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

type PubKeyTransfer struct {
	description string
	targetHost  string
	pubKeyPath  string
	opts        PubKeyTransferOptions
}

type PubKeyTransferOptions struct {
	WithMockCommand ssh.CommandFunc
}

func NewPubKeyTransfer(description string, targetHost string, privKeyPath string, opts PubKeyTransferOptions) *PubKeyTransfer {
	return &PubKeyTransfer{description: description, targetHost: targetHost, pubKeyPath: privKeyPath + ".pub", opts: opts}
}

func (kt *PubKeyTransfer) Description() string {
	return kt.description
}

func (kt *PubKeyTransfer) buildTransferConnection(stdin []byte) *target.Connection {
	opts := target.ConnectionOptions{WithLoginShell: true, WithStdin: stdin}

	if kt.opts.WithMockCommand != nil {
		opts.WithMockCommand = kt.opts.WithMockCommand
	}

	conn := target.NewConnection(kt.targetHost, opts)

	return &conn
}

func (kt *PubKeyTransfer) Run(outputWriter io.Writer) error {
	pubKey, err := os.ReadFile(kt.pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key %s: %w", kt.pubKeyPath, err)
	}

	conn := kt.buildTransferConnection(pubKey)
	cmdOutput, err := conn.Run(remoteAuthorizedKeysCommand)
	if err != nil {
		return fmt.Errorf("failed to transfer public key to target %s: %w", kt.targetHost, err)
	}
	_, err = outputWriter.Write([]byte(cmdOutput))
	return err
}

func (kt *PubKeyTransfer) DryRun(output io.Writer) error {
	conn := kt.buildTransferConnection(nil)
	if err := conn.DryRun(remoteAuthorizedKeysCommand, output); err != nil {
		return fmt.Errorf("failed to write dry-run output for public key transfer to target %s: %w", kt.targetHost, err)
	}
	return nil
}
