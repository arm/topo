package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arm/topo/internal/template"
	"github.com/stretchr/testify/require"
)

const TestSshTarget = "test-target"

const LsCpuOutputRaw = `{
	"lscpu": [
		{"field": "Vendor ID:", "data": "ARM"},
		{"field": "Model name:", "data": "Cortex-A55"},
		{"field": "Core(s) per cluster:", "data": "2"},
		{"field": "Socket(s):", "data": "-"},
		{"field": "Cluster(s):", "data": "1"},
		{"field": "Flags:", "data": "fp asimd"}
	]
}`

func RequireDocker(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found. Install Docker: https://docs.docker.com/desktop/")
	}
}

func RequireLinuxDockerEngine(t testing.TB) {
	t.Helper()
	RequireDocker(t)
	cmd := exec.Command("docker", "info", "--format", "{{.OSType}}")
	output, err := cmd.Output()
	require.NoError(t, err, "failed to get docker info")
	if strings.TrimSpace(string(output)) != "linux" {
		t.Skip("skipping test that requires linux docker engine")
	}
}

func RequireOS(t testing.TB, os string) {
	t.Helper()
	if runtime.GOOS != os {
		t.Skipf("skipping test that requires %s", os)
	}
}

func RequireWriteFile(t testing.TB, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}

func WithFakeSSH(t testing.TB, env map[string]string, fn func(argsFile string)) {
	t.Helper()

	dir := t.TempDir()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve testutil path")
	scriptPath := filepath.Join(filepath.Dir(file), "testdata", "fake_ssh.sh")
	script, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "read fake ssh script")
	sshPath := filepath.Join(dir, "ssh")
	require.NoError(t, os.WriteFile(sshPath, script, 0o755), "write fake ssh")

	argsFile := filepath.Join(dir, "args.txt")
	t.Setenv("SSH_TEST_ARGS_FILE", argsFile)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for k, v := range env {
		t.Setenv(k, v)
	}

	fn(argsFile)
}

func ReadArgsLines(t testing.TB, argsFile string) []string {
	t.Helper()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		require.NoError(t, err, "read args")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func SanitiseTestName(t testing.TB) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ",", "")
	return name
}

func WriteComposeFile(t *testing.T, dir, content string) string {
	t.Helper()
	composePath := filepath.Join(dir, template.ComposeFilename)
	RequireWriteFile(t, composePath, content)
	return composePath
}
