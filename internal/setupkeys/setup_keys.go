package setupkeys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	goperation "github.com/arm/topo/internal/deploy/operation"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
)

func NewKeySetup(target string, privKeyPath string) (goperation.Sequence, error) {
	if privKeyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to determine default key path: %w", err)
		}

		keyName := fmt.Sprintf("id_ed25519_topo_%s", slugifyTarget(target))
		privKeyPath = filepath.Join(home, ".ssh", keyName)
	}

	ops := []goperation.Operation{
		sshkeygen.NewSSHKeyGen("Generate SSH key pair for target", target, "ed25519", privKeyPath, sshkeygen.SSHKeyGenOptions{}),
		pubkeytransfer.NewPubKeyTransfer("Transfer public key to target and set it as an authorized key", target, privKeyPath, pubkeytransfer.PubKeyTransferOptions{}),
	}
	return goperation.NewSequence(ops...), nil
}

func slugifyTarget(target string) string {
	var b strings.Builder
	for _, r := range target {
		toWrite := '_'
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			toWrite = r
		}

		b.WriteRune(toWrite)
	}

	return b.String()
}
