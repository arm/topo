package setupkeys

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
)

type KeyType string

const (
	KeyTypeED25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
)

func NewKeySetup(target string, privKeyPath string, keyType string) (operation.Sequence, error) {
	parsedKeyType, err := ParseKeyType(keyType)
	if err != nil {
		return nil, err
	}

	ops := []operation.Operation{
		sshkeygen.NewSSHKeyGen("Generate SSH key pair for target", target, string(parsedKeyType), privKeyPath, sshkeygen.SSHKeyGenOptions{}),
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

func ParseKeyType(s string) (KeyType, error) {
	switch KeyType(s) {
	case KeyTypeED25519:
		return KeyTypeED25519, nil
	case KeyTypeRSA:
		return KeyTypeRSA, nil
	default:
		return "", fmt.Errorf("unsupported key type %q, supported types: %s, %s", s, KeyTypeED25519, KeyTypeRSA)
	}
}
