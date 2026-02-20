package setup_keys

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

// Exported only so blackbox tests can inject fakes.
type ExecCommandFunc func(string, ...string) *exec.Cmd

var (
	ExecCommand ExecCommandFunc = exec.Command
	SSHExec                     = ssh.Exec
)

func CreateKeyPair(targetHost string, targetFileName string, privKeyPath string, cmdOutput io.Writer, dryRun bool) (string, error) {
	if privKeyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine home directory: %w", err)
		}

		keyName := fmt.Sprintf("id_ed25519_topo_%s", targetFileName)
		privKeyPath = filepath.Join(home, ".ssh", keyName)
	}

	if err := ensureDir(privKeyPath); err != nil {
		return "", err
	}

	keyPairCreationCmd := ExecCommand("ssh-keygen", "-t", "ed25519", "-f", privKeyPath, "-C", targetHost)

	if dryRun && cmdOutput != nil {
		_, err := fmt.Fprintln(cmdOutput, keyPairCreationCmd.String())
		if err != nil {
			return "", err
		}
	} else if !dryRun {
		if cmdOutput != nil {
			keyPairCreationCmd.Stdout = cmdOutput
			keyPairCreationCmd.Stderr = cmdOutput
		}

		if err := keyPairCreationCmd.Run(); err != nil {
			return "", fmt.Errorf("command %q failed: %w", keyPairCreationCmd.String(), err)
		}
	}
	return privKeyPath, nil
}

func TransferPubKey(targetHost string, privKeyPath string, output io.Writer, dryRun bool) error {
	pubKeyPath := privKeyPath + ".pub"

	if dryRun {
		_, err := fmt.Fprintf(output, "ssh %s %q < %s.pub\n", targetHost, remoteAuthorizedKeysCommand, privKeyPath)
		return err
	}

	pubKey, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key %s: %w", pubKeyPath, err)
	}

	opts := target.ConnectionOptions{WithLoginShell: true, WithStdin: pubKey}
	pubKeyTransfer := target.NewConnection(targetHost, SSHExec, opts)
	if _, err := pubKeyTransfer.Run(remoteAuthorizedKeysCommand); err != nil {
		return err
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

func SanitizeTarget(target string) string {
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
