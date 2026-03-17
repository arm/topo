package testutil

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

var (
	RequireDocker            = gtestutil.RequireDocker
	RequireLinuxDockerEngine = gtestutil.RequireLinuxDockerEngine
	RequireWriteFile         = gtestutil.RequireWriteFile
	SanitiseTestName         = gtestutil.SanitiseTestName
)

func TestImageName(t *testing.T) string {
	return "test-image-" + gtestutil.SanitiseTestName(t)
}

func TestContainerName(t *testing.T) string {
	return "test-container-" + gtestutil.SanitiseTestName(t)
}

func RequireImageExists(t *testing.T, h ssh.Host, imageName string) {
	t.Helper()
	inspectCmd := command.Docker(h, "image", "inspect", imageName)
	output, err := inspectCmd.CombinedOutput()
	require.NoError(t, err, "image %s doesn't exist: %s output: %s", imageName, command.String(inspectCmd), string(output))
}

func BuildMinimalImage(t *testing.T, h ssh.Host, imageName string) {
	t.Helper()
	dockerfileContent := `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`
	buildCmd := command.Docker(h, "build", "-t", imageName, "-")
	buildCmd.Stdin = bytes.NewBufferString(dockerfileContent)
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "failed to build image %s: %s output: %s", imageName, command.String(buildCmd), string(output))

	RequireImageExists(t, h, imageName)
	t.Cleanup(func() {
		removeCmd := command.Docker(h, "image", "rm", "-f", imageName)
		if err := removeCmd.Run(); err != nil {
			t.Logf("failed to remove image %s: %v", imageName, err)
		}
	})
}
