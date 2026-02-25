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

func TestEnsureProjectIsLinuxArm64Ready(t *testing.T) {
	cases := []struct {
		name        string
		compose     string
		wantErr     bool
		errContains string
	}{
		{
			name: "allows remoteproc runtime",
			compose: `
services:
  rtos-firmware:
    build:
      context: .
    runtime: io.containerd.remoteproc.v1
`,
		},
		{
			name: "allows linux arm64 without variant",
			compose: `
services:
  app:
    image: alpine
    platform: linux/arm64
`,
		},
		{
			name: "allows linux arm64 with variant",
			compose: `
services:
  app:
    image: alpine
    platform: linux/arm64/v8
`,
		},
		{
			name: "fails when platform missing",
			compose: `
services:
  api:
    image: busybox
`,
			wantErr:     true,
			errContains: "missing platform declaration",
		},
		{
			name: "fails when platform mismatch",
			compose: `
services:
  api:
    image: busybox
    platform: linux/amd64
`,
			wantErr:     true,
			errContains: "linux/amd64",
		},
		{
			name: "skips remoteproc runtime",
			compose: `
services:
  firmware:
    image: zephyr
    runtime: io.containerd.remoteproc.v1
`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			composeFile := writeComposeFile(t, tc.compose)
			err := checks.EnsureProjectIsLinuxArm64Ready(composeFile)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
				return
			}
			require.NoError(t, err)
		})
	}
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

func TestEnsureSSHGatewayPortsAreDisabled(t *testing.T) {
	t.Run("skips localhost", func(t *testing.T) {
		logResult := checks.EnsureSSHGatewayPortsAreDisabled(ssh.PlainLocalhost, target.ConnectionOptions{WithLoginShell: true})
		require.Empty(t, logResult)
	})

	cases := []struct {
		name        string
		target      ssh.Host
		responses   map[string]mockedCommand
		wantEmpty   bool
		wantLevel   string
		wantMessage string
	}{
		{
			name:   "openssh allows when gateway ports disabled",
			target: ssh.Host("halloween@opensshRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm": {output: "sshd\n", exitCode: 0},
				"sshd -T":     {output: "gatewayports no\n", exitCode: 0},
			},
			wantEmpty: true,
		},
		{
			name:   "openssh fails when gateway ports enabled",
			target: ssh.Host("christmas@opensshRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm": {output: "sshd\n", exitCode: 0},
				"sshd -T":     {output: "gatewayports yes\nInclude /etc/ssh/sshd_config.d/*.conf", exitCode: 0},
			},
			wantLevel:   "ERROR",
			wantMessage: "SSH GatewayPorts must be disabled on christmas@opensshRemote",
		},
		{
			name:   "openssh fails when gateway ports are clientspecified",
			target: ssh.Host("easter@opensshRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm": {output: "sshd\n", exitCode: 0},
				"sshd -T":     {output: "gatewayports clientspecified\nInclude /etc/ssh/sshd_config.d/*.conf", exitCode: 0},
			},
			wantLevel:   "ERROR",
			wantMessage: "SSH GatewayPorts must be disabled on easter@opensshRemote",
		},
		{
			name:   "openssh falls back to config file",
			target: ssh.Host("boxingday@opensshRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm":              {output: "sshd\n", exitCode: 0},
				"sshd -T":                  {output: "", exitCode: 1},
				"cat /etc/ssh/sshd_config": {output: "gatewayports no\n", exitCode: 0},
			},
			wantEmpty: true,
		},
		{
			name:   "openssh warns when config is ambiguous",
			target: ssh.Host("mayday@opensshRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm":              {output: "sshd\n", exitCode: 0},
				"sshd -T":                  {output: "", exitCode: 1},
				"cat /etc/ssh/sshd_config": {output: "Include /etc/ssh/sshd_config.d/*.conf\ngatewayports no", exitCode: 0},
			},
			wantLevel:   "WARN",
			wantMessage: "SSH GatewayPorts setting on mayday@opensshRemote is uncertain due to Match or Include directives in sshd_config",
		},
		{
			name:   "openssh propagates read error",
			target: ssh.Host("bonfire@opensshRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm":              {output: "sshd\n", exitCode: 0},
				"sshd -T":                  {output: "", exitCode: 1},
				"cat /etc/ssh/sshd_config": {output: "", exitCode: 1},
			},
			wantLevel:   "ERROR",
			wantMessage: "failed to read sshd_config on bonfire@opensshRemote; unable to confirm SSH GatewayPorts is disabled. Error:",
		},
		{
			name:   "fails when dropbear uses -a",
			target: ssh.Host("teddy@DropbearRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm":            {output: "dropbear\n", exitCode: 0},
				"ps aux | grep dropbear": {output: "/usr/sbin/dropbear -R -E -a\n", exitCode: 0},
			},
			wantLevel:   "ERROR",
			wantMessage: "SSH GatewayPorts must be disabled on teddy@DropbearRemote: Dropbear detected with -a option",
		},
		{
			name:   "allows dropbear without -a",
			target: ssh.Host("fluffy@DropbearRemote"),
			responses: map[string]mockedCommand{
				"ps -eo comm":            {output: "dropbear\n", exitCode: 0},
				"ps aux | grep dropbear": {output: "/usr/sbin/dropbear -R -E\n", exitCode: 0},
			},
			wantEmpty: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockExec := mockExecByCommand(t, tc.responses)
			logResult := checks.EnsureSSHGatewayPortsAreDisabled(tc.target, target.ConnectionOptions{WithMockExec: mockExec})
			if tc.wantEmpty {
				require.Empty(t, logResult)
				return
			}
			require.NotEmpty(t, logResult)
			if tc.wantLevel != "" {
				require.Equal(t, tc.wantLevel, string(logResult[0].Level))
			}
			if tc.wantMessage != "" {
				require.Contains(t, logResult[0].Message, tc.wantMessage)
			}
		})
	}
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
