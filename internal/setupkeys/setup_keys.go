package setupkeys

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
)

const (
	KeyTypeED25519 string = "ed25519"
	KeyTypeRSA     string = "rsa"
)

func NewKeySetup(target string, privKeyPath string, keyType string) (operation.Sequence, error) {
	if err := isValidKeyType(keyType); err != nil {
		return nil, err
	}

	ops := []operation.Operation{
		sshkeygen.NewSSHKeyGen("Generate SSH key pair for target", target, keyType, privKeyPath, sshkeygen.SSHKeyGenOptions{}),
		pubkeytransfer.NewPubKeyTransfer("Transfer public key to target and set it as an authorized key", target, privKeyPath, pubkeytransfer.PubKeyTransferOptions{}),
	}
	return operation.NewSequence(ops...), nil
}

func GetDefaultPrivateKeyPath(targetSlug string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine default key path: %w", err)
	}
	keyName := fmt.Sprintf("id_ed25519_topo_%s", targetSlug)
	privKeyPath := filepath.Join(home, ".ssh", keyName)
	return privKeyPath, nil
}

func isValidKeyType(s string) error {
	switch s {
	case KeyTypeED25519, KeyTypeRSA:
		return nil
	default:
		return fmt.Errorf("unsupported key type %q, supported types: %s, %s", s, KeyTypeED25519, KeyTypeRSA)
	}
}
