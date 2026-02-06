package core

import (
	"os/exec"
	"testing"
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
			out = DefaultBoardHostname
		case 1:
			out = psOut
		case 2:
			out = inspectOut
		}
		call++
		return exec.Command("echo", out)
	}
	items, err := ReadContainersInfo(DefaultSshTarget)
	if err != nil {
		t.Fatalf("ReadContainersInfo: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}
