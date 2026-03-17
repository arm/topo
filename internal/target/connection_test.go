package target_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	t.Run("run executes command successfully", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string, _ []byte, _ ...string) *exec.Cmd {
			return testutil.CmdWithOutput("success", 0)
		}
		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockExec: mockExec})

		out, err := conn.Run("ls")

		assert.NoError(t, err)
		assert.Equal(t, "success", out)
	})

	t.Run("run returns error", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string, _ []byte, _ ...string) *exec.Cmd {
			return testutil.CmdWithOutput("", 1)
		}
		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockExec: mockExec})

		out, err := conn.Run("ls")

		assert.Error(t, err)
		assert.Empty(t, out)
	})

	t.Run("run with mutliplexing enabled includes Control args", func(t *testing.T) {
		testutil.RequireOS(t, "linux")
		var capturedArgs string
		mockExec := func(_ ssh.Host, _ string, _ []byte, sshArgs ...string) *exec.Cmd {
			capturedArgs = strings.Join(sshArgs, " ")
			return testutil.CmdWithOutput("success", 0)
		}
		conn := target.NewConnection("hostname", target.ConnectionOptions{Multiplex: true, WithMockExec: mockExec})

		_, err := conn.Run("ls")

		assert.NoError(t, err)
		assert.True(t, strings.Contains(capturedArgs, "-o ControlMaster"), "missing ControlMaster argument")
		assert.True(t, strings.Contains(capturedArgs, "-o ControlPersist"), "missing ControlPersist argument")
		assert.True(t, strings.Contains(capturedArgs, "-o ControlPath"), "missing ControlPath argument")
	})

	t.Run("run with mutliplexing enabled does not include Control args on windows", func(t *testing.T) {
		testutil.RequireOS(t, "windows")
		var capturedArgs string
		mockExec := func(_ ssh.Host, _ string, _ []byte, sshArgs ...string) *exec.Cmd {
			capturedArgs = strings.Join(sshArgs, " ")
			return testutil.CmdWithOutput("success", 0)
		}
		conn := target.NewConnection("hostname", target.ConnectionOptions{Multiplex: true, WithMockExec: mockExec})

		_, err := conn.Run("ls")

		assert.NoError(t, err)
		assert.False(t, strings.Contains(capturedArgs, "-o ControlMaster"), "unexpected ControlMaster argument")
		assert.False(t, strings.Contains(capturedArgs, "-o ControlPersist"), "unexpected ControlPersist argument")
		assert.False(t, strings.Contains(capturedArgs, "-o ControlPath"), "unexpected ControlPath argument")
	})
}

func TestBinaryExists(t *testing.T) {
	t.Run("when binary found returns nil", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string, _ []byte, _ ...string) *exec.Cmd {
			return testutil.CmdWithOutput("/foo/bar", 0)
		}
		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockExec: mockExec})

		assert.NoError(t, conn.BinaryExists("bar"))
	})

	t.Run("invalid format returns an error", func(t *testing.T) {
		mockExec := func(_ ssh.Host, _ string, _ []byte, _ ...string) *exec.Cmd {
			return testutil.CmdWithOutput("/foo/bar", 0)
		}
		conn := target.NewConnection("hostname", target.ConnectionOptions{WithMockExec: mockExec})

		assert.Error(t, conn.BinaryExists("b a r"))
	})
}
