package deploy_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestDeployment(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Run", func(t *testing.T) {
		container := testutil.StartContainer(t, testutil.DinDContainer)

		t.Run("builds images, transfers them, and starts services", func(t *testing.T) {
			remoteDockerHost := ssh.NewDestination(container.SSHDestination)
			tmpDir := t.TempDir()
			dockerFilePath := filepath.Join(tmpDir, "Dockerfile")
			dockerFileContent := `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`
			testutil.RequireWriteFile(t, dockerFilePath, dockerFileContent)
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := fmt.Sprintf(`
name: %s
services:
  busybox:
    image: busybox
    command: ["tail", "-f", "/dev/null"]
  a-service:
    build: .
`, testutil.TestProjectName(t))
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			t.Cleanup(func() { testutil.ForceComposeDown(t, composeFilePath) })

			deployOpts := deploy.DeployOptions{TargetHost: remoteDockerHost}

			err := deploy.Deploy(context.Background(), os.Stdout, composeFilePath, deployOpts)

			require.NoError(t, err)
			testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)
		})
	})
}
