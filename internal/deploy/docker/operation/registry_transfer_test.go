package operation_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/testutil"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryTransfer(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			transfer := operation.NewRegistryTransfer("any.yaml", ssh.PlainLocalhost, ssh.PlainLocalhost, operation.DefaultRegistryPort)

			got := transfer.Description()

			assert.Equal(t, "Transfer via registry", got)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("it prints registry transfer commands", func(t *testing.T) {
			testutil.RequireDocker(t)
			var buf bytes.Buffer
			h := ssh.PlainLocalhost
			port := operation.DefaultRegistryPort
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := `
services:
  alpine:
    image: alpine:latest
  nginx:
    image: nginx:latest
`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			transfer := operation.NewRegistryTransfer(composeFilePath, h, ssh.Host("user@remote"), port)

			err := transfer.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()

			alpineTag := fmt.Sprintf("localhost:%s/alpine:latest", port)
			nginxTag := fmt.Sprintf("localhost:%s/nginx:latest", port)

			expected := strings.TrimSpace(fmt.Sprintf(`
docker tag alpine:latest %[1]s
docker push %[1]s
docker -H ssh://user@remote pull %[1]s
docker -H ssh://user@remote tag %[1]s alpine:latest
docker tag nginx:latest %[2]s
docker push %[2]s
docker -H ssh://user@remote pull %[2]s
docker -H ssh://user@remote tag %[2]s nginx:latest
`, alpineTag, nginxTag)) + "\n"

			assert.Equal(t, expected, got)
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("it transfers images via registry", func(t *testing.T) {
			testutil.RequireLinuxDockerEngine(t)
			h := ssh.PlainLocalhost
			port := operation.DefaultRegistryPort
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			dockerFilePath := filepath.Join(tmpDir, "Dockerfile")
			imageName := testutil.TestImageName(t)
			composeFileContent := fmt.Sprintf(`
services:
  test:
    build: .
    image: %s
`, imageName)
			dockerFileContent := `FROM alpine:latest`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			testutil.RequireWriteFile(t, dockerFilePath, dockerFileContent)

			buildCmd := command.DockerCompose(h, composeFilePath, "build")
			buildOut, err := buildCmd.CombinedOutput()
			require.NoError(t, err, "build failed: %s", string(buildOut))

			rmCmd := command.Docker(h, "rm", "-f", operation.RegistryContainerName)
			rmOut, rmErr := rmCmd.CombinedOutput()
			if rmErr != nil {
				t.Logf("registry container cleanup (expected if not running): %s", string(rmOut))
			}

			startReg := command.Docker(h, "run", "-d", "--restart=always", "-p", fmt.Sprintf("%s:5000", port), "--name", operation.RegistryContainerName, "registry:2")
			startOut, err := startReg.CombinedOutput()
			require.NoError(t, err, "could not start registry for test: %s", string(startOut))
			t.Cleanup(func() {
				rmReg := command.Docker(h, "rm", "-f", operation.RegistryContainerName)
				_ = rmReg.Run()
			})

			transfer := operation.NewRegistryTransfer(composeFilePath, h, h, port)
			err = transfer.Run(os.Stdout)
			require.NoError(t, err)
			testutil.RequireImageExists(t, h, imageName)
		})
	})
}
