package e2e

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestInstall(t *testing.T) {
	target := testutil.StartTargetContainer(t)
	topo := buildBinary(t)

	t.Run("installs the binary", func(t *testing.T) {
		targetURL := fmt.Sprintf("ssh://%s", target.SSHConnectionString)

		out, err := installRemoteprocRuntime(topo, targetURL)

		require.NoError(t, err, out)
		requireInstalled(t, "remoteproc-runtime", targetURL)
	})

	t.Run("replaces the binary when installing a new version", func(t *testing.T) {
		targetURL := fmt.Sprintf("ssh://%s", target.SSHConnectionString)
		requireInstalled(t, "remoteproc-runtime", targetURL)

		out, err := installRemoteprocRuntime(topo, targetURL)

		require.NoError(t, err, out)
		requireInstalled(t, "remoteproc-runtime", targetURL)
	})
}

func installRemoteprocRuntime(topo string, targetURL string) (string, error) {
	args := []string{"install", "remoteproc-runtime", "--target", targetURL}
	installCmd := exec.Command(topo, args...)

	out, err := installCmd.CombinedOutput()
	return string(out), err
}

func requireInstalled(t *testing.T, binary, targetURL string) {
	verifyCmd := exec.Command(
		"ssh",
		targetURL,
		"command -v",
		binary,
		">/dev/null && echo ok",
	)

	vout, verr := verifyCmd.CombinedOutput()

	require.NoError(t, verr, "verify failed: %s\noutput:\n%s", verifyCmd.String(), vout)
	require.Contains(t, string(vout), "ok")
}
