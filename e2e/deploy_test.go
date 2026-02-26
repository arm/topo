package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	target := testutil.StartTargetContainer(t)
	topo := buildBinary(t)

	t.Run("Init, add and Deploy", func(t *testing.T) {
		projectDir := t.TempDir()
		composeFile := filepath.Join(projectDir, "compose.yaml")
		t.Cleanup(func() {
			composeDown(t, composeFile, target.SSHConnectionString)
		})

		requireInit(t, topo, projectDir)
		require.FileExists(t, composeFile)
		nameArgValue := "Topo"
		expectedResponse := fmt.Sprintf("Hello %s\n", nameArgValue)

		requireExtend(t, topo, projectDir, composeFile, nameArgValue)

		requireDeploy(t, topo, projectDir, target.SSHConnectionString)
		port, err := testutil.GetContainerPublicPort(target.ContainerName, "8080")
		require.NoError(t, err)
		assertResponseBody(t, fmt.Sprintf("http://localhost:%s/", port), expectedResponse)
	})

	t.Run("Clone and deploy", func(t *testing.T) {
		baseDir := t.TempDir()
		cloneDir := filepath.Join(baseDir, "project")
		composeFile := filepath.Join(cloneDir, "compose.yaml")
		t.Cleanup(func() {
			composeDown(t, composeFile, target.SSHConnectionString)
		})

		nameArgValue := "Topo"
		requireClone(t, topo, baseDir, cloneDir, "testdata/services/hello-server", fmt.Sprintf("NAME=%s", nameArgValue))
		requireDeploy(t, topo, cloneDir, target.SSHConnectionString)
		expectedResponse := fmt.Sprintf("Hello %s\n", nameArgValue)
		port, err := testutil.GetContainerPublicPort(target.ContainerName, "8080")
		require.NoError(t, err)
		assertResponseBody(t, fmt.Sprintf("http://localhost:%s/", port), expectedResponse)
	})
}

func requireClone(t *testing.T, topo string, projectDir string, cloneDir string, remoteDir string, extraArgs ...string) {
	remoteDirPath, err := filepath.Abs(remoteDir)
	require.NoError(t, err)
	cloneDirPath, err := filepath.Abs(cloneDir)
	require.NoError(t, err)

	args := []string{"clone", cloneDirPath, fmt.Sprintf("dir:%s", remoteDirPath)}
	args = append(args, extraArgs...)
	cloneCmd := exec.Command(topo, args...)
	cloneCmd.Dir = projectDir
	out, err := cloneCmd.CombinedOutput()

	require.NoErrorf(t, err, "clone failed: %s", out)
}

func requireInit(t *testing.T, topo, projectDir string) {
	initCmd := exec.Command(topo, "init")
	initCmd.Dir = projectDir

	out, err := initCmd.CombinedOutput()

	require.NoErrorf(t, err, "init failed: %s", out)
}

func requireExtend(t *testing.T, topo, projectDir, composeFile, customName string) {
	templateDir, err := filepath.Abs("testdata/services/hello-server")
	require.NoError(t, err)
	extendCmd := exec.Command(topo, "extend", composeFile,
		fmt.Sprintf("dir:%s", templateDir), "--",
		fmt.Sprintf("NAME=%s", customName))
	extendCmd.Dir = projectDir

	out, err := extendCmd.CombinedOutput()

	require.NoErrorf(t, err, "extend failed: %s", out)
}

func requireDeploy(t *testing.T, topo, projectDir, sshTarget string, extraArgs ...string) {
	args := []string{"deploy", "--target", sshTarget, "--skip-project-checks"}
	args = append(args, extraArgs...)

	deployCmd := exec.Command(topo, args...)
	deployCmd.Dir = projectDir

	out, err := deployCmd.CombinedOutput()

	require.NoErrorf(t, err, "deploy failed: %s", out)
}

func assertResponseBody(t *testing.T, url, wantBody string) {
	var resp *http.Response
	require.Eventually(t, func() bool {
		var err error
		resp, err = http.Get(url)
		if err != nil {
			return false
		}
		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			return false
		}
		return true
	}, 30*time.Second, 1*time.Second, "service did not become healthy")
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, wantBody, string(body))
}

func composeDown(t *testing.T, composeFile, sshTarget string) {
	t.Helper()
	cmd := exec.Command("docker", "-H", "ssh://"+sshTarget, "compose", "-f", composeFile, "down", "-v")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("compose down failed: %v, output: %s", err, out)
	}
}
