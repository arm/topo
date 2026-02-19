package setupkeys

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	goperation "github.com/arm/topo/internal/deploy/operation"
)

func NewKeyCreationAndPlacementOnTarget(target string, keyPath string) (goperation.Sequence, error) {
	if keyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to determine home directory: %w", err)
		}

		keyName := fmt.Sprintf("id_ed25519_topo_%s", sanitizeTarget(target))
		keyPath = filepath.Join(home, ".ssh", keyName)
	}

	if err := ensureDir(keyPath); err != nil {
		return nil, err
	}

	ops := []goperation.Operation{
		newSetupKeysOperation("Generate SSH key pair for target", []string{"ssh-keygen", "-t", "ed25519", "-f", keyPath, "-C", target}),
		newSetupKeysOperation("Copy SSH public key to target", []string{"ssh-copy-id", "-i", keyPath + ".pub", target}),
	}
	return goperation.NewSequence(ops...), nil
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

type setupKeysOperation struct {
	description string
	command     []string
}

func newSetupKeysOperation(description string, command []string) *setupKeysOperation {
	return &setupKeysOperation{description: description, command: command}
}

func (c *setupKeysOperation) Description() string {
	return c.description
}

func (c *setupKeysOperation) Run(cmdOutput io.Writer) error {
	if len(c.command) == 0 {
		return fmt.Errorf("no command configured")
	}
	cmd := exec.Command(c.command[0], c.command[1:]...)
	if cmdOutput != nil {
		cmd.Stdout = cmdOutput
		cmd.Stderr = cmdOutput
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %q failed: %w", strings.Join(c.command, " "), err)
	}
	return nil
}

func (c *setupKeysOperation) DryRun(output io.Writer) error {
	if output == nil {
		return nil
	}
	_, err := fmt.Fprintln(output, strings.Join(c.command, " "))
	return err
}
