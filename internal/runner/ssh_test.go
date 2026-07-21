package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForDirect(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX shell script")
	}

	binDir := t.TempDir()
	fakeSSH := filepath.Join(binDir, "ssh")
	err := os.WriteFile(fakeSSH, []byte("#!/bin/sh\nfor arg do command=$arg; done\nprintf '%s' \"$command\"\n"), 0o700)
	require.NoError(t, err)
	t.Setenv("PATH", binDir)
	r := runner.ForDirect(ssh.NewDestination("example.com"))

	stdout, _, err := r.Run(context.Background(), "echo $PATH")

	require.NoError(t, err)
	assert.Equal(t, "echo $PATH", stdout)
}
