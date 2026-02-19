package setupkeys

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
)

const remoteAuthorizedKeysCommand = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

var execCommand = exec.Command

func NewKeyPairCreation(targetHost string, privKeyPath string) (*exec.Cmd, string, error) {
	if privKeyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", fmt.Errorf("failed to determine home directory: %w", err)
		}

		keyName := fmt.Sprintf("id_ed25519_topo_%s", sanitizeTarget(targetHost))
		privKeyPath = filepath.Join(home, ".ssh", keyName)
	}

	if err := ensureDir(privKeyPath); err != nil {
		return nil, "", err
	}

	return execCommand("ssh-keygen", "-t", "ed25519", "-f", privKeyPath, "-C", targetHost), privKeyPath, nil
}

func RunKeyPairCreation(keyPairCreationCmd *exec.Cmd, cmdOutput io.Writer, dryRun bool) error {
	if dryRun && cmdOutput != nil {
		_, err := fmt.Fprintln(cmdOutput, keyPairCreationCmd.String())
		if err != nil {
			return err
		}
	} else if !dryRun {
		if cmdOutput != nil {
			keyPairCreationCmd.Stdout = cmdOutput
			keyPairCreationCmd.Stderr = cmdOutput
		}

		if err := keyPairCreationCmd.Run(); err != nil {
			return fmt.Errorf("command %q failed: %w", keyPairCreationCmd.String(), err)
		}
	}

	return nil
}

func NewPubKeyTransfer(targetHost string, privKeyPath string, dryRun bool) (*target.Connection, error) {
	pubKeyPath := privKeyPath + ".pub"
	opts := target.ConnectionOptions{WithLoginShell: true}
	if !dryRun {
		pubKey, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key %s: %w", pubKeyPath, err)
		}
		opts.WithStdin = pubKey
	}

	pubKeyTransfer := target.NewConnection(targetHost, ssh.Exec, opts)
	return &pubKeyTransfer, nil
}

func RunPubKeyTransfer(pubKeyTransfer *target.Connection, privKeyPath string, output io.Writer, dryRun bool) error {
	if dryRun {
		_, err := fmt.Fprintf(output, "ssh %s %q < %s.pub\n", pubKeyTransfer.SSHTarget, remoteAuthorizedKeysCommand, privKeyPath)
		return err
	} else {
		if _, err := pubKeyTransfer.Run(remoteAuthorizedKeysCommand); err != nil {
			return err
		}
	}
	return nil
}

func ensureDir(keyPath string) error {
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	return nil
}

func sanitizeTarget(target string) string {
	var b strings.Builder
	for _, r := range target {
		toWrite := '_'
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			toWrite = r
		}

		b.WriteRune(toWrite)
	}

	sanitized := b.String()
	return sanitized
}
