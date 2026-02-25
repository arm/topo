package sshconfig_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arm/topo/internal/setupkeys/sshconfig"
	"github.com/stretchr/testify/require"
)

func TestModifySSHConfigWritesIncludeAndFragment(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
		if vol := filepath.VolumeName(tmp); vol != "" {
			t.Setenv("HOMEDRIVE", vol)
			t.Setenv("HOMEPATH", strings.TrimPrefix(tmp, vol))
		}
	}

	targetHost := "user@example.com:2222"
	targetFileName := "user_example_com_2222"
	privKeyPath := filepath.Join(tmp, ".ssh", fmt.Sprintf("id_ed25519_topo_%s", targetFileName))

	err := sshconfig.ModifySSHConfig(targetHost, privKeyPath, targetFileName, false, nil)
	require.NoError(t, err)

	mainConfigPath := filepath.Join(tmp, ".ssh", "config")
	mainConfig, err := os.ReadFile(mainConfigPath)
	require.NoError(t, err)
	mainConfigText := string(mainConfig)
	require.Contains(t, mainConfigText, "Include ")
	require.Contains(t, mainConfigText, filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config", "*.conf")))

	fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))
	fragment, err := os.ReadFile(fragmentPath)
	require.NoError(t, err)

	expected := "" +
		"Host example.com\n" +
		"  HostName example.com\n" +
		"  User user\n" +
		"  Port 2222\n" +
		fmt.Sprintf("  IdentityFile %s\n", filepath.ToSlash(privKeyPath)) +
		"  IdentitiesOnly yes\n"
	require.Equal(t, expected, string(fragment))
}
