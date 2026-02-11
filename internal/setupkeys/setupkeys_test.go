package setupkeys

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewSequenceDryRunOutputsCommands(t *testing.T) {
	testutil.RequireOS(t, "linux")

	t.Run("default key path", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)

		seq, err := NewKeyCreationAndPlacementOnTarget("user@example.com", "")
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, seq.DryRun(&buf))

		keyPath := filepath.Join(tmp, ".ssh", "id_ed25519_topo_user_example.com")
		wantKeygen := "ssh-keygen -t ed25519 -f " + keyPath + " -C user@example.com"
		wantCopy := "ssh-copy-id -i " + keyPath + ".pub user@example.com"
		got := buf.String()
		require.Contains(t, got, wantKeygen, "DryRun output should include keygen command")
		require.Contains(t, got, wantCopy, "DryRun output should include ssh-copy-id command")
	})

	t.Run("custom key path", func(t *testing.T) {
		keyPath := filepath.Join(t.TempDir(), "custom_keys", "id_ed25519_custom")

		seq, err := NewKeyCreationAndPlacementOnTarget("user@example.com", keyPath)
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, seq.DryRun(&buf))

		wantKeygen := "ssh-keygen -t ed25519 -f " + keyPath + " -C user@example.com"
		wantCopy := "ssh-copy-id -i " + keyPath + ".pub user@example.com"
		got := buf.String()
		require.Contains(t, got, wantKeygen, "DryRun output should include keygen command")
		require.Contains(t, got, wantCopy, "DryRun output should include ssh-copy-id command")
	})
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
			if got := sanitizeTarget(tt.input); got != tt.want {
				t.Fatalf("sanitizeTarget(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
