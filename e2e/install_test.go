package e2e

import (
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestInstall(t *testing.T) {
	target := testutil.StartTargetContainer(t)
	topo := buildBinary(t)

	t.Run("installs the binary", func(t *testing.T) {
		out, err := installRemoteprocRuntime(topo, target.SSHDestination)

		require.NoError(t, err, out)
		requireInstalled(t, "remoteproc-runtime", target.SSHDestination)
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
