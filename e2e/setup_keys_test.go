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

	Step(t, "health reports unknown host key and suggests trusting it through SSH")
	out := runTopo(t, topo, "health", "--target", container.SSHDestination)
	assert.Contains(t, out, "✗ Connectivity")
	assert.Contains(t, out, "host key is unknown")
	assert.Contains(t, out, "Verify and trust the target's SSH host key")
	assert.Contains(t, out, fmt.Sprintf("ssh -o StrictHostKeyChecking=ask '%s'", container.SSHDestination))

	Step(t, "trust the test host through SSH")
	askpass := writeAskPassScript(t, sshRootPassword)
	trustCmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=accept-new", container.SSHDestination, "true")
	trustCmd.Env = append(os.Environ(), "SSH_ASKPASS="+askpass, "SSH_ASKPASS_REQUIRE=force")
	trustOut, err := trustCmd.CombinedOutput()
	require.NoError(t, err, "ssh failed: %s", trustOut)

	Step(t, "health reports authentication failure and suggests setup-keys")
	out = runTopo(t, topo, "health", "--target", container.SSHDestination)
	assert.Contains(t, out, "✗ Connectivity")
	assert.Contains(t, out, "authentication failed")
	assert.Contains(t, out, "Configure SSH keys on remote target")
	assert.Contains(t, out, fmt.Sprintf("topo setup-keys --target %s", container.SSHDestination))

	Step(t, "setup-keys generates keys and installs them on the target")
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
	out = runTopo(t, topo, "health", "--target", container.SSHDestination, "--verbose")
	assert.Contains(t, out, "✓ Connectivity")
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
