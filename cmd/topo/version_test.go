package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionFlagReportsLinkedVersion(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)

	binaryName := "topo"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)

	buildCommand := exec.Command(
		"go",
		"build",
		"-ldflags",
		"-X github.com/arm/topo/internal/version.Version=4.1.0 -X github.com/arm/topo/internal/version.GitCommit=deadbeef",
		"-o",
		binaryPath,
		".",
	)
	buildCommand.Dir = workingDirectory

	buildOutput, err := buildCommand.CombinedOutput()
	require.NoError(t, err, string(buildOutput))

	versionCommand := exec.Command(binaryPath, "--version")
	versionOutput, err := versionCommand.CombinedOutput()
	require.NoError(t, err, string(versionOutput))

	assert.Equal(t, "topo version 4.1.0 (commit: deadbeef)\n", string(versionOutput))
}
