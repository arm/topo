package describe_test

import (
	"testing"

	"github.com/arm/topo/internal/describe"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
