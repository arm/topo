package core

import (
	"os/exec"
	"testing"

	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadContainersInfo_MockExec(t *testing.T) {
	orig := ExecCommand
	defer func() { ExecCommand = orig }()
	psOut := `{"Command":"cmd","CreatedAt":"now","ID":"id1","Image":"img1","Labels":"","LocalVolumes":"","Mounts":"","Names":"svc1","Networks":"n","Ports":"","RunningFor":"","Size":"","State":"running","Status":"Up"}`
	inspectOut := `{};runc`
	call := 0
	ExecCommand = func(name string, args ...string) *exec.Cmd {
		var out string
		switch call {
		case 0:
			out = psOut
		case 1:
			out = inspectOut
		}
		call++
		return exec.Command("echo", out)
	}
	items, err := ReadContainersInfo(testutil.TestSshTarget)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}
