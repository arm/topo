package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractArmFeatures(t *testing.T) {
	t.Run("extracts mapped ARM features and ignores unrecognised", func(t *testing.T) {
		target := Target{
			features: []string{"fp", "asimd", "sve2", "sme"},
		}
		res := extractArmFeatures(target)
		expected := []string{"NEON", "SVE2", "SME"}
		assert.Equal(t, expected, res)
	})

	t.Run("returns empty slice if no matching features", func(t *testing.T) {
		target := Target{features: []string{"fp", "crc32"}}
		res := extractArmFeatures(target)
		assert.Empty(t, res)
	})
}

func TestHealthCheckStringBuilder(t *testing.T) {
	t.Run("shows all fields when ssh and connected", func(t *testing.T) {
		hostPath := []string{"ssh"}
		target := Target{
			connectionError: nil,
			features:        []string{"asimd", "sve"},
		}

		output := HealthCheckStringBuilder(hostPath, target)
		assert.Contains(t, output, "SSH: ✅")
		assert.Contains(t, output, "Connected: ✅")
		assert.Contains(t, output, "Features (Linux Host): NEON, SVE")
	})

	t.Run("shows ❌ when ssh not present", func(t *testing.T) {
		hostPath := []string{}
		target := Target{connectionError: nil}

		output := HealthCheckStringBuilder(hostPath, target)
		assert.Contains(t, output, "SSH: ❌")
		assert.NotContains(t, output, "Connected: ✅") // no target section shown
	})

	t.Run("shows ❌ when target connection failed", func(t *testing.T) {
		hostPath := []string{"ssh"}
		target := Target{connectionError: assert.AnError}

		output := HealthCheckStringBuilder(hostPath, target)
		assert.Contains(t, output, "Connected: ❌")
	})
}
