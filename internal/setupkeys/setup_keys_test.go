package setupkeys_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewKeySetupDryRun(t *testing.T) {
	tests := []struct {
		name         string
		keyType      string
		wantTarget   string
		inputKeyPath string
		wantKeyPath  string
	}{
		{
			name:         "default key path",
			keyType:      "ed25519",
			inputKeyPath: "",
			wantTarget:   "user@some1thing.com",
			wantKeyPath:  filepath.Join(".ssh", "id_ed25519_topo_user_some1thing.com"),
		},
		{
			name:         "custom key path",
			keyType:      "ed25519",
			inputKeyPath: filepath.Join("custom_keys", "id_ed25519_custom"),
			wantTarget:   "user@some2thing.com",
			wantKeyPath:  filepath.Join("custom_keys", "id_ed25519_custom"),
		},
		{
			name:         "rsa key type",
			keyType:      "rsa",
			inputKeyPath: filepath.Join("custom_keys", "id_rsa_custom"),
			wantTarget:   "user@some3thing.com",
			wantKeyPath:  filepath.Join("custom_keys", "id_rsa_custom"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testutil.SetHomeDir(t, tmp)

			wantKeyPath := tt.wantKeyPath
			if tt.inputKeyPath == "" {
				wantKeyPath = filepath.Join(tmp, tt.wantKeyPath)
			}
			targetSlug := ssh.Host(tt.wantTarget).Slugify()
			inputKeyPath := tt.inputKeyPath
			if inputKeyPath == "" {
				var err error
				inputKeyPath, err = setupkeys.GetDefaultPrivateKeyPath(targetSlug)
				require.NoError(t, err)
			}
			got, err := setupkeys.NewKeySetup(tt.wantTarget, inputKeyPath, tt.keyType)
			require.NoError(t, err)
			require.Len(t, got, 2)
			require.IsType(t, &sshkeygen.SSHKeyGen{}, got[0])
			require.IsType(t, &pubkeytransfer.PubKeyTransfer{}, got[1])
			require.Equal(t, "Generate SSH key pair for target", got[0].Description())
			require.Equal(t, "Transfer public key to target and set it as an authorized key", got[1].Description())
			var buf bytes.Buffer
			require.NoError(t, got.DryRun(&buf))

			wantKeygen := "ssh-keygen -t " + tt.keyType + " -f " + wantKeyPath + " -C " + tt.wantTarget
			wantSSH := "ssh -- " + tt.wantTarget
			wantCmd := "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
			require.Contains(t, buf.String(), wantKeygen, "DryRun output should include keygen command")
			require.Contains(t, buf.String(), wantSSH, "DryRun output should include ssh command with correct target")
			require.Contains(t, buf.String(), wantCmd, "DryRun output should include correct authorized key addition command to be executed on target")
		})
	}
}

func TestNewKeySetupUnsupportedKeyType(t *testing.T) {
	_, err := setupkeys.ParseKeyType("ecdsa")
	require.EqualError(t, err, "unsupported key type \"ecdsa\", supported types: ed25519, rsa")
}
