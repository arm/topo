package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupKeysJourney(t *testing.T) {
	container := testutil.StartContainer(t, testutil.PasswordedSSHContainer)
	topo := buildBinary(t)

	Step(t, "health reports unknown host key and suggests accept-new-host-keys")
	out := runTopo(t, topo, "health", "--target", container.SSHDestination)
	assert.Contains(t, out, "Connectivity: ❌ (ssh host key is unknown)")
	assert.Contains(t, out, "Trust the target's SSH host key")

	Step(t, "health with accept-new-host-keys trusts host and suggests setup-keys")
	out = runTopo(t, topo, "health", "--target", container.SSHDestination, "--accept-new-host-keys")
	assert.Contains(t, out, "Connectivity: ❌ (ssh authentication failed)")
	assert.Contains(t, out, "Configure SSH keys on remote target")

	Step(t, "setup-keys generates keys and installs them on the target")
	askpass := writeAskPassScript(t, sshRootPassword)
	cmd := exec.Command(topo, "setup-keys", "--target", container.SSHDestination)
	cmd.Env = append(os.Environ(), []string{
		"SSH_ASKPASS=" + askpass,
		"SSH_ASKPASS_REQUIRE=force",
	}...)
	setupOut, err := cmd.CombinedOutput()
	require.NoError(t, err, "topo failed: %s", setupOut)
	assert.Contains(t, string(setupOut), "Generate SSH key pair")
	assert.Contains(t, string(setupOut), "Transfer public key")

	Step(t, "healthcheck is successful")
	out = runTopo(t, topo, "health", "--target", container.SSHDestination)
	assert.Contains(t, out, "Connectivity: ✅")
}

func runTopo(t *testing.T, topo string, args ...string) string {
	t.Helper()
	cmd := exec.Command(topo, args...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "topo failed: %s", out)
	return string(out)
}

const sshRootPassword = "topo-test"

func writeAskPassScript(t *testing.T, password string) string {
	t.Helper()
	dir := t.TempDir()

	if runtime.GOOS == "windows" {
		script := fmt.Sprintf(`@echo off
echo %%~1 | findstr /i "assphrase" >nul && (echo.) || (echo %s)
`, password)
		path := filepath.Join(dir, "askpass.bat")
		require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
		return path
	}

	script := fmt.Sprintf(`#!/bin/sh
case "$1" in
  *assphrase*) echo "" ;;
  *) echo "%s" ;;
esac
`, password)
	path := filepath.Join(dir, "askpass.sh")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}
