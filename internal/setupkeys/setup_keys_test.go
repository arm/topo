package setupkeys_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewKeyCreateAndPlaceSequenceDryRun(t *testing.T) {
	tests := []struct {
		name         string
		wantTarget   string
		inputKeyPath string
		wantKeyPath  string
	}{
		{
			name:         "default key path",
			inputKeyPath: "",
			wantTarget:   "user@some1thing.com",
			wantKeyPath:  filepath.Join(".ssh", "id_ed25519_topo_user_some1thing.com"),
		},
		{
			name:         "custom key path",
			inputKeyPath: filepath.Join("custom_keys", "id_ed25519_custom"),
			wantTarget:   "user@some2thing.com",
			wantKeyPath:  filepath.Join("custom_keys", "id_ed25519_custom"),
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
			got, err := setupkeys.NewKeyCreateAndPlaceSequence(tt.wantTarget, tt.inputKeyPath)
			require.NoError(t, err)
			require.Len(t, got, 2)
			require.IsType(t, &sshkeygen.SSHKeyGen{}, got[0])
			require.IsType(t, &pubkeytransfer.PubKeyTransfer{}, got[1])
			require.Equal(t, "Generate SSH key pair for target", got[0].Description())
			require.Equal(t, "Transfer public key to target and set it as an authorized key", got[1].Description())
			var buf bytes.Buffer
			require.NoError(t, got.DryRun(&buf))

			wantKeygen := "ssh-keygen -t ed25519 -f " + wantKeyPath + " -C " + tt.wantTarget
			wantSSH := "ssh -- " + tt.wantTarget
			wantCmd := "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
			require.Contains(t, buf.String(), wantKeygen, "DryRun output should include keygen command")
			require.Contains(t, buf.String(), wantSSH, "DryRun output should include ssh command with correct target")
			require.Contains(t, buf.String(), wantCmd, "DryRun output should include correct authorized key addition command to be executed on target")
		})
	}
}

func TestSlugifyTarget(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user@example.com", "user_example.com"},
		{"Example-Host", "Example-Host"},
		{"spaces and/tabs", "spaces_and_tabs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, setupkeys.SlugifyTarget(tt.input), "SlugifyTarget should replace special characters with underscores and keep allowed characters")
		})
	}
}
