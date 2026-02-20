package setup_keys_test

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arm/topo/internal/setup_keys"
	"github.com/stretchr/testify/require"
)

func TestKeyPairCreationAndPlacementOnTargetDryRun(t *testing.T) {
	tests := []struct {
		name         string
		inputKeyPath string
		wantKeyPath  string
	}{
		{
			name:         "default key path",
			inputKeyPath: "",
			wantKeyPath:  filepath.Join(".ssh", "id_ed25519_topo_user_example.com"),
		},
		{
			name:         "custom key path",
			inputKeyPath: filepath.Join("custom_keys", "id_ed25519_custom"),
			wantKeyPath:  filepath.Join("custom_keys", "id_ed25519_custom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			if runtime.GOOS == "windows" {
				t.Setenv("USERPROFILE", tmp)
				if vol := filepath.VolumeName(tmp); vol != "" {
					t.Setenv("HOMEDRIVE", vol)
					t.Setenv("HOMEPATH", strings.TrimPrefix(tmp, vol))
				}
			}

			inputKeyPath := tt.inputKeyPath
			if inputKeyPath != "" {
				inputKeyPath = filepath.Join(tmp, inputKeyPath)
			}

			var buf bytes.Buffer
			targetFileName := setup_keys.SanitizeTarget("user@example.com")
			keyPath, err := setup_keys.CreateKeyPair("user@example.com", targetFileName, inputKeyPath, &buf, true)
			require.NoError(t, err)

			wantKeyPath := filepath.Join(tmp, tt.wantKeyPath)
			require.Equal(t, wantKeyPath, keyPath, "NewKeyPairCreation should return expected key path")

			wantKeygen := "-t ed25519 -f " + keyPath + " -C user@example.com"
			got := buf.String()
			require.Contains(t, got, "ssh-keygen", "DryRun output should include keygen command")
			require.Contains(t, got, wantKeygen, "DryRun output should include keygen arguments")

			transferErr := setup_keys.TransferPubKey("user@example.com", keyPath, &buf, true)
			require.NoError(t, transferErr)
			got = buf.String()
			require.Contains(t, got, "ssh user@example.com", "DryRun output should include ssh command")
			require.Contains(t, got, keyPath+".pub", "DryRun output should include public key path")
		})
	}
}

func TestSanitizeTarget(t *testing.T) {
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
			if got := setup_keys.SanitizeTarget(tt.input); got != tt.want {
				t.Fatalf("sanitizeTarget(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
