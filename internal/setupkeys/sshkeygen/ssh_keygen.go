package sshkeygen

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

type execSSHKeyGen func(keyType string, keyPath string, targetHost string) *exec.Cmd

type SSHKeyGen struct {
	description string
	dest        ssh.Destination
	keyType     string
	keyPath     string
	exec        execSSHKeyGen
}

type SSHKeyGenOptions struct {
	WithMockKeyGen execSSHKeyGen
}

func NewSSHKeyGen(description string, dest ssh.Destination, keyType string, keyPath string, opts SSHKeyGenOptions) *SSHKeyGen {
	execFn := command.SSHKeyGen
	if opts.WithMockKeyGen != nil {
		execFn = opts.WithMockKeyGen
	}

	return &SSHKeyGen{
		description: description,
		dest:        dest,
		keyType:     keyType,
		keyPath:     keyPath,
		exec:        execFn,
	}
}

func (kg *SSHKeyGen) Description() string {
	return kg.description
}

func (kg *SSHKeyGen) buildCommand() *exec.Cmd {
	return kg.exec(kg.keyType, kg.keyPath, kg.dest.String())
}

func (kg *SSHKeyGen) Run(cmdOutput io.Writer) error {
	dir := filepath.Dir(kg.keyPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	cmd := kg.buildCommand()
	cmd.Stdin = os.Stdin
	cmd.Stdout = cmdOutput
	cmd.Stderr = cmdOutput
	return cmd.Run()
}
