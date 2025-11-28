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
	composeFile := filepath.Join(projectDir, "compose.project.yaml")
	require.FileExists(t, composeFile)

	nameArgValue := "Topo"
	requireAddService(t, topo, projectDir, composeFile, "my-hello-service", nameArgValue)

	requireDeploy(t, topo, projectDir, vm.SSHConnectionString)
	expectedResponse := fmt.Sprintf("Hello %s\n", nameArgValue)
	assertResponseBody(t, "http://localhost:8080/", expectedResponse)
}

func requireInit(t *testing.T, topo, projectDir string) {
	initCmd := exec.Command(topo, "init")
	initCmd.Dir = projectDir

	out, err := initCmd.CombinedOutput()

	require.NoErrorf(t, err, "init failed: %s", out)
}

func requireAddService(t *testing.T, topo, projectDir, composeFile, serviceName, customName string) {
	templateDir, err := filepath.Abs("testdata/services/hello-server")
	require.NoError(t, err)
	serviceAddCmd := exec.Command(topo, "service", "add", composeFile, serviceName,
		fmt.Sprintf("dir:%s", templateDir), "--no-prompt", "--",
		fmt.Sprintf("NAME=%s", customName))
	serviceAddCmd.Dir = projectDir

	out, err := serviceAddCmd.CombinedOutput()

	require.NoErrorf(t, err, "service add failed: %s", out)
}

func requireDeploy(t *testing.T, topo, projectDir, sshTarget string) {
	deployCmd := exec.Command(topo, "deploy", "--target", sshTarget)
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
			resp.Body.Close()
			return false
		}
		return true
	}, 30*time.Second, 1*time.Second, "service did not become healthy")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, wantBody, string(body))
}
