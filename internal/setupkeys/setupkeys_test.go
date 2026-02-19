package setupkeys

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewKeyCreationAndPlacementOnTarget(t *testing.T) {
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

func TestKeyCreationAndPlacementOnTarget(t *testing.T) {
	testutil.RequireOS(t, "linux")

	keyPath := filepath.Join(t.TempDir(), "custom_keys", "id_ed25519_custom_run")
	fakeBinDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "commands.log")

	writeFakeSSHCommand(t, fakeBinDir, "ssh-keygen", logFile)
	writeFakeSSHCommand(t, fakeBinDir, "ssh-copy-id", logFile)

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+originalPath)

	seq, err := NewKeyCreationAndPlacementOnTarget("user@example.com", keyPath)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, seq.Run(&buf))
	require.Contains(t, buf.String(), "ssh-keygen invoked", "Run output should include fake ssh-keygen output")
	require.Contains(t, buf.String(), "ssh-copy-id invoked", "Run output should include fake ssh-copy-id output")

	logData, err := os.ReadFile(logFile)
	require.NoError(t, err)
	log := string(logData)
	require.Contains(t, log, fmt.Sprintf("ssh-keygen -t ed25519 -f %s -C user@example.com", keyPath))
	require.Contains(t, log, fmt.Sprintf("ssh-copy-id -i %s.pub user@example.com", keyPath))
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

func writeFakeSSHCommand(t *testing.T, dir, name, logFile string) {
	t.Helper()
	script := fmt.Sprintf(`#!/bin/sh
echo "%s invoked"
echo "$0 $@" >> %s
`, name, logFile)
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(script), 0o700))
}
