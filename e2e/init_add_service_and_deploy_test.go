package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAddServiceAndDeploy(t *testing.T) {
	testutil.RequireDocker(t)
	vm := testutil.StartDockerVM(t)
	topo := buildBinary(t)

	projectDir := t.TempDir()

	requireInit(t, topo, projectDir)
	composeFile := filepath.Join(projectDir, "compose.yaml")
	require.FileExists(t, composeFile)
	nameArgValue := "Topo"
	expectedResponse := fmt.Sprintf("Hello %s\n", nameArgValue)

	requireExtend(t, topo, projectDir, composeFile, nameArgValue)

	requireDeploy(t, topo, projectDir, vm.SSHConnectionString)
	assertResponseBody(t, "http://localhost:8080/", expectedResponse)
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
		fmt.Sprintf("dir:%s", templateDir), "--no-prompt", "--",
		fmt.Sprintf("NAME=%s", customName))
	extendCmd.Dir = projectDir

	out, err := extendCmd.CombinedOutput()

	require.NoErrorf(t, err, "extend failed: %s", out)
}

func requireDeploy(t *testing.T, topo, projectDir, sshTarget string, extraArgs ...string) {
	args := []string{"deploy", "--target", sshTarget}
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
