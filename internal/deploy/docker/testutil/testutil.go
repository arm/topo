package testutil

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	RequireDocker            = gtestutil.RequireDocker
	RequireLinuxDockerEngine = gtestutil.RequireLinuxDockerEngine
	RequireWriteFile         = gtestutil.RequireWriteFile
	SanitiseTestName         = gtestutil.SanitiseTestName
	StartTargetContainer     = gtestutil.StartTargetContainer
)

func TestImageName(t *testing.T) string {
	return "test-image-" + gtestutil.SanitiseTestName(t)
}

func TestContainerName(t *testing.T) string {
	return "test-container-" + gtestutil.SanitiseTestName(t)
}

func TestProjectName(t *testing.T) string {
	return "test-project-" + gtestutil.SanitiseTestName(t)
}

func RequireImageExists(t *testing.T, h ssh.Destination, imageName string) {
	t.Helper()
	inspectCmd := command.Docker(h, "image", "inspect", imageName)
	output, err := inspectCmd.CombinedOutput()
	require.NoError(t, err, "image %s doesn't exist: %s output: %s", imageName, command.String(inspectCmd), string(output))
}

func BuildMinimalImage(t *testing.T, h ssh.Destination, imageName string) {
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

func ForceComposeDown(t *testing.T, composeFilePath string) {
	t.Helper()
	// #nosec G204 -- ignore as its a test helper
	err := exec.Command("docker", "compose", "-f", composeFilePath, "down", "-v").Run()
	if err != nil {
		t.Logf("docker compose down failed: %v (compose file: %s)", err, composeFilePath)
	}
}

func AssertContainersRunning(t *testing.T, h ssh.Destination, composeFilePath string) {
	t.Helper()
	dockerCmd := command.DockerCompose(h, composeFilePath, "ps", "--format", "json")
	output, err := dockerCmd.CombinedOutput()
	require.NoError(t, err, string(output))

	require.NotEmpty(t, bytes.TrimSpace(output), "no containers running")

	containers, err := unmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "running", container["State"], "container %s is not running: %s", container["Name"], container["State"])
	}
}

func AssertContainersStopped(t *testing.T, h ssh.Destination, composeFilePath string) {
	t.Helper()
	dockerCmd := command.DockerCompose(h, composeFilePath, "ps", "--format", "json", "--all")
	output, err := dockerCmd.CombinedOutput()
	require.NoError(t, err, string(output))

	require.NotEmpty(t, bytes.TrimSpace(output), "no containers reported")

	containers, err := unmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "exited", container["State"], "expected container %s to be exited (state=%s)", container["Name"], container["State"])
	}
}

type jsonObject map[string]any

func unmarshalNDJSON(ndJSON []byte) ([]jsonObject, error) {
	objects := []jsonObject{}
	lines := bytes.SplitSeq(bytes.TrimSpace(ndJSON), []byte("\n"))

	for line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var obj jsonObject
		err := json.Unmarshal(line, &obj)
		if err != nil {
			return objects, err
		}
		objects = append(objects, obj)
	}

	return objects, nil
}
