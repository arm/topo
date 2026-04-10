package describe_test

import (
	"testing"

	"github.com/arm/topo/internal/describe"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/assert/yaml"
	"github.com/stretchr/testify/require"
)

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

		content := testutil.RequireReadFile(t, outputFile)
		err = yaml.Unmarshal([]byte(content), &reportOut)
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

		content := testutil.RequireReadFile(t, outputFile2)
		require.Equal(t, outputFile1, outputFile2)
		assert.Contains(t, content, "feature1")
		assert.NotContains(t, content, "feature2")
	})
}

func TestReadTargetDescriptionFromFile(t *testing.T) {
	t.Run("Correctly reads and parses target description from yaml file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := dir + "/target-description.yaml"
		content := `host:
  - model: Cortex-A55
    features:
      - asimd
      - sve
    cores: 2
remoteprocs:
  - name: remoteproc0
totalmemory_kb: 16384
`
		testutil.RequireWriteFile(t, filePath, content)
		profile, err := describe.ReadTargetDescriptionFromFile(filePath)

		require.NoError(t, err)
		assert.Equal(t, target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{
					Model:    "Cortex-A55",
					Features: []string{"asimd", "sve"},
					Cores:    2,
				},
			},
			RemoteCPU:     []target.RemoteprocCPU{{Name: "remoteproc0"}},
			TotalMemoryKb: 16384,
		}, *profile)
	})

	t.Run("returns error when target description file does not exist", func(t *testing.T) {
		_, err := describe.ReadTargetDescriptionFromFile("/no/such/file.yaml")

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read target description file")
	})

	t.Run("returns error when target description yaml is invalid", func(t *testing.T) {
		dir := t.TempDir()
		filePath := dir + "/invalid.yaml"
		testutil.RequireWriteFile(t, filePath, "host_processor: [")

		_, err := describe.ReadTargetDescriptionFromFile(filePath)

		assert.ErrorContains(t, err, "failed to parse target description file")
	})
}
