package setupkeys

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
)

func NewKeySetup(target string, privKeyPath string) (operation.Sequence, error) {
	ops := []operation.Operation{
		sshkeygen.NewSSHKeyGen("Generate SSH key pair for target", target, "ed25519", privKeyPath, sshkeygen.SSHKeyGenOptions{}),
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
