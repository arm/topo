package describe_test

import (
	"os"
	"strings"
	"testing"

	"github.com/arm/topo/internal/describe"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/assert/yaml"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Run("returns hardware profile for given target", func(t *testing.T) {
		mockExecSSH := func(target ssh.Host, command string) (string, error) {
			if command == "lscpu --json" {
				return testutil.LsCpuOutputRaw, nil
			}
			if strings.Contains(command, "remoteproc") {
				return "remoteproc1 remoteproc2", nil
			}
			return "", nil
		}
		expected := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{
					ModelName: "Cortex-A55",
					Features:  []string{"fp", "asimd"},
					Cores:     2,
				},
			},
			RemoteCPU: []target.RemoteprocCPU{
				{Name: "remoteproc1"},
				{Name: "remoteproc2"},
			},
		}

		conn := target.NewConnection("test", mockExecSSH, target.ConnectionOptions{})
		report, err := describe.GenerateTargetDescription(conn)

		require.NoError(t, err)
		assert.Equal(t, expected, report)
	})

	t.Run("fails if ssh commands cannot be executed", func(t *testing.T) {
		mockExecSSH := func(target ssh.Host, command string) (string, error) {
			return "", assert.AnError
		}

		conn := target.NewConnection("test", mockExecSSH, target.ConnectionOptions{})
		_, err := describe.GenerateTargetDescription(conn)

		assert.Error(t, err)
	})
}

func TestWriteTargetDescriptionFile(t *testing.T) {
	t.Run("writes full target to description to given directory", func(t *testing.T) {
		dir := t.TempDir()
		report := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"feature1", "feature2"}},
			},
			RemoteCPU: []target.RemoteprocCPU{
				{Name: "remoteproc1"},
				{Name: "remoteproc2"},
			},
		}
		var reportOut target.HardwareProfile

		outputFile, err := describe.WriteTargetDescriptionToFile(dir, report)
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		err = yaml.Unmarshal(content, &reportOut)
		require.NoError(t, err)
		require.FileExists(t, outputFile)
		assert.Equal(t, report, reportOut)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		dir := t.TempDir()
		report1 := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"feature1", "feature2"}},
			},
		}
		report2 := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"feature1"}},
			},
		}

		outputFile1, err := describe.WriteTargetDescriptionToFile(dir, report1)
		require.NoError(t, err)
		outputFile2, err := describe.WriteTargetDescriptionToFile(dir, report2)
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile2)
		require.NoError(t, err)
		require.Equal(t, outputFile1, outputFile2)
		assert.Contains(t, string(content), "feature1")
		assert.NotContains(t, string(content), "feature2")
	})
}
