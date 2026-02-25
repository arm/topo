package checks_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	checks "github.com/arm/topo/internal/deploy/project_checks"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestEnsureProjectIsLinuxArm64Ready_SucceedsWithValidRemoteProc(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  rtos-firmware:
    build:
      context: .
    runtime: io.containerd.remoteproc.v1
`)

	require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
}

func TestEnsureProjectIsLinuxArm64Ready_SucceedsWithValidPlatforms(t *testing.T) {
	t.Run("without variant", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  app:
    image: alpine
    platform: linux/arm64
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})

	t.Run("with variant", func(t *testing.T) {
		composeFile := writeComposeFile(t, `
services:
  app:
    image: alpine
    platform: linux/arm64/v8
`)

		require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
	})
}

func TestEnsureProjectIsLinuxArm64Ready_FailsWhenPlatformMissing(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  api:
    image: busybox
`)

	err := checks.EnsureProjectIsLinuxArm64Ready(composeFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing platform declaration")
}

func TestEnsureProjectIsLinuxArm64Ready_FailsWhenPlatformMismatch(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  api:
    image: busybox
    platform: linux/amd64
`)

	err := checks.EnsureProjectIsLinuxArm64Ready(composeFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "linux/amd64")
}

func TestEnsureProjectIsLinuxArm64Ready_SkipsRemoteprocRuntime(t *testing.T) {
	composeFile := writeComposeFile(t, `
services:
  firmware:
    image: zephyr
    runtime: io.containerd.remoteproc.v1
`)

	require.NoError(t, checks.EnsureProjectIsLinuxArm64Ready(composeFile))
}

func writeComposeFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}
	return path
}

func TestEnsureSSHGatewayPortsAreDisabled_SkipsLocalhost(t *testing.T) {
	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.PlainLocalhost, target.ConnectionOptions{WithLoginShell: true})
	require.Empty(t, logResult)
}

func TestEnsureSSHGatewayPortsAreDisabled_AllowsNoinOpenSSH(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm": {output: "sshd\n", exitCode: 0},
		"sshd -T":     {output: "gatewayports no\n", exitCode: 0},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("pumpkin@halloweenRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.Empty(t, logResult)
}

func TestEnsureSSHGatewayPortsAreDisabled_FailsWhenEnabledinOpenSSH(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm": {output: "sshd\n", exitCode: 0},
		"sshd -T":     {output: "gatewayports yes\nInclude /etc/ssh/sshd_config.d/*.conf", exitCode: 0},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("turkey@xmasRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.NotEmpty(t, logResult)
	require.Equal(t, string(logResult[0].Level), "ERROR")
	require.Contains(t, logResult[0].Message, "SSH GatewayPorts must be disabled on turkey@xmasRemote")
}

func TestEnsureSSHGatewayPortsAreDisabled_FailsWhenClientSpecifiedinOpenSSH(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm": {output: "sshd\n", exitCode: 0},
		"sshd -T":     {output: "gatewayports clientspecified\nInclude /etc/ssh/sshd_config.d/*.conf", exitCode: 0},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("sunday@brunchRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.NotEmpty(t, logResult)
	require.Equal(t, string(logResult[0].Level), "ERROR")
	require.Contains(t, logResult[0].Message, "SSH GatewayPorts must be disabled on sunday@brunchRemote")
}

func TestEnsureSSHGatewayPortsAreDisabled_FallsBackToConfigFileinOpenSSH(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm":              {output: "sshd\n", exitCode: 0},
		"sshd -T":                  {output: "", exitCode: 1},
		"cat /etc/ssh/sshd_config": {output: "gatewayports no\n", exitCode: 0},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("pancake@shroveRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.Empty(t, logResult)
}

func TestEnsureSSHGatewayPortsAreDisabled_WarnsWhenConfigIsAmbiguousinOpenSSH(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm":              {output: "sshd\n", exitCode: 0},
		"sshd -T":                  {output: "", exitCode: 1},
		"cat /etc/ssh/sshd_config": {output: "Include /etc/ssh/sshd_config.d/*.conf\ngatewayports no", exitCode: 0},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("mardi@grasRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.NotEmpty(t, logResult)
	require.Equal(t, string(logResult[0].Level), "WARN")
	require.Contains(t, logResult[0].Message, "SSH GatewayPorts setting on mardi@grasRemote is uncertain due to Match or Include directives in sshd_config")
}

func TestEnsureSSHGatewayPortsAreDisabled_PropagatesReadErrorinOpenSSH(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm":              {output: "sshd\n", exitCode: 0},
		"sshd -T":                  {output: "", exitCode: 1},
		"cat /etc/ssh/sshd_config": {output: "", exitCode: 1},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("pancake@shroveRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.NotEmpty(t, logResult)
	require.Equal(t, string(logResult[0].Level), "ERROR")
	require.Contains(t, logResult[0].Message, "failed to read sshd_config on pancake@shroveRemote; unable to confirm SSH GatewayPorts is disabled. Error:")
}

func TestEnsureSSHGatewayPortsAreDisabled_dashADetectionErrorDropbear(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm":            {output: "dropbear\n", exitCode: 0},
		"ps aux | grep dropbear": {output: "/usr/sbin/dropbear -R -E -a\n", exitCode: 0},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("teddy@DropbearRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.NotEmpty(t, logResult)
	require.Equal(t, string(logResult[0].Level), "ERROR")
	require.Contains(t, logResult[0].Message, "SSH GatewayPorts must be disabled on teddy@DropbearRemote: Dropbear detected with -a option")
}

func TestEnsureSSHGatewayPortsAreDisabled_AllowswithoutDashADropbear(t *testing.T) {
	mockExec := mockExecByCommand(t, map[string]mockedCommand{
		"ps -eo comm":            {output: "dropbear\n", exitCode: 0},
		"ps aux | grep dropbear": {output: "/usr/sbin/dropbear -R -E\n", exitCode: 0},
	})

	logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.Host("teddy@DropbearRemote"), target.ConnectionOptions{WithMockExec: mockExec})
	require.Empty(t, logResult)
}

type mockedCommand struct {
	output   string
	exitCode int
}

func mockExecByCommand(t *testing.T, responses map[string]mockedCommand) func(ssh.Host, string, []byte, ...string) *exec.Cmd {
	t.Helper()
	return func(_ ssh.Host, command string, _ []byte, _ ...string) *exec.Cmd {
		response, ok := responses[command]
		if !ok {
			t.Fatalf("unexpected command: %s", command)
		}
		return testutil.CmdWithOutput(response.output, response.exitCode)
	}
}
