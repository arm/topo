package setupkeys

import (
	"fmt"
	"path/filepath"

	"github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
	"github.com/arm/topo/internal/ssh"
)

type KeyType string

const (
	KeyTypeED25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
)

func NewKeySetup(dest ssh.Destination, privKeyPath string, keyType KeyType) (operation.Sequence, error) {
	sshRunner := runner.NewSSH(dest, runner.SSHOptions{})
	ops := []operation.Operation{
		sshkeygen.NewSSHKeyGen("Generate SSH key pair for target", dest, string(keyType), privKeyPath, sshkeygen.SSHKeyGenOptions{}),
		pubkeytransfer.NewPubKeyTransfer(privKeyPath, sshRunner),
	}
	return operation.NewSequence(ops...), nil
}

func GetDefaultPrivateKeyPath(sshDir string, targetSlug string) (string, error) {
	keyName := fmt.Sprintf("id_ed25519_topo_%s", targetSlug)
	privKeyPath := filepath.Join(sshDir, keyName)
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
