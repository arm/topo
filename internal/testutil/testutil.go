package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const TestSshTarget = "test-target"

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

func RequirePodman(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman is not installed")
	}
	if _, err := exec.LookPath("docker-compose"); err != nil {
		t.Skip("docker-compose is not installed")
	}
	if output, err := exec.Command("podman", "info").CombinedOutput(); err != nil {
		t.Skipf("local Podman engine is unavailable: %v: %s", err, output)
	}
}

func RequireOS(t testing.TB, os ...string) {
	t.Helper()
	if !slices.Contains(os, runtime.GOOS) {
		t.Skipf("skipping test that requires %s", os)
	}
}

func RequireWriteFile(t testing.TB, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)
}

func RequireMkdirAll(t testing.TB, path string) {
	t.Helper()
	err := os.MkdirAll(path, 0o700)
	require.NoError(t, err)
}

func SanitiseTestName(t testing.TB) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ",", "")
	return name
}

func RequireWriteComposeFile(t testing.TB, dir, content string) string {
	t.Helper()
	composePath := filepath.Join(dir, compose.DefaultFileName())
	RequireWriteFile(t, composePath, content)
	return composePath
}

// FixPodmanInDockerQuirk avoids a Docker Desktop nested-container restriction.
// The Podman target inherits oom_score_adj: 200, but Podman otherwise starts
// each service with oom_score_adj: 0. Docker Desktop rejects that decrease, so
// this adds oom_score_adj: 200 to each fixture service.
func FixPodmanInDockerQuirk(contents string) (string, error) {
	var definition map[string]any
	if err := yaml.Unmarshal([]byte(contents), &definition); err != nil {
		return "", err
	}
	services, ok := definition["services"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("Compose file services must be a mapping")
	}
	for name, value := range services {
		service, ok := value.(map[string]any)
		if !ok {
			return "", fmt.Errorf("service %q must be a mapping", name)
		}
		service["oom_score_adj"] = 200
	}

	updatedContents, err := yaml.Marshal(definition)
	if err != nil {
		return "", err
	}
	return string(updatedContents), nil
}

func CmdWithStderr(output string, exitCode int) *exec.Cmd {
	if runtime.GOOS == "windows" {
		script := "$OutputEncoding = [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; [Console]::Error.Write($env:TOPO_CMD_OUT); exit [int]$env:TOPO_CMD_CODE"
		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		cmd.Env = append(os.Environ(), "TOPO_CMD_OUT="+output, fmt.Sprintf("TOPO_CMD_CODE=%d", exitCode))
		return cmd
	}
	// #nosec G204 -- ignore as its a test helper
	return exec.Command("sh", "-c", fmt.Sprintf("printf %%s \"$1\" >&2; exit %d", exitCode), "sh", output)
}

func CmdWithOutput(output string, exitCode int) *exec.Cmd {
	if runtime.GOOS == "windows" {
		// PowerShell: emit exact bytes (no extra newline), UTF-8, and requested exit code.
		script := "$OutputEncoding = [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; [Console]::Out.Write($env:TOPO_CMD_OUT); exit [int]$env:TOPO_CMD_CODE"
		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		cmd.Env = append(os.Environ(), "TOPO_CMD_OUT="+output, fmt.Sprintf("TOPO_CMD_CODE=%d", exitCode))
		return cmd
	}
	// #nosec G204 -- ignore as its a test helper
	return exec.Command("sh", "-c", fmt.Sprintf("printf %%s \"$1\"; exit %d", exitCode), "sh", output)
}

func RequireReadFile(t testing.TB, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func RequireEnvFileValues(t testing.TB, path string, want map[string]string) {
	t.Helper()
	got, err := env.ReadFile(path)
	require.NoError(t, err, "failed to load env file %s", path)
	require.Equal(t, want, got, "env file %s", path)
}

func AssertFileContents(t *testing.T, wantContents string, path string) {
	t.Helper()

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, wantContents, string(got))
}

func AssertJsonGoldenFile(t *testing.T, got string, goldenPath string) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		err := os.WriteFile(goldenPath, []byte(got), 0o644)
		require.NoError(t, err)
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	want := string(wantBytes)

	require.JSONEq(t, want, got, "output did not match golden file %s", goldenPath)
}
