package upgrade_test

import (
	"testing"

	"github.com/arm/topo/internal/upgrade"
	"github.com/stretchr/testify/assert"
)

func TestGetUpgradeCommand(t *testing.T) {
	t.Run("homebrew managed topo is upgradeable via brew", func(t *testing.T) {
		isManagedByUs, cmd := upgrade.GetUpgradeCommand("/opt/homebrew/Cellar/topo/4.1.0/bin/topo")

		assert.False(t, isManagedByUs)
		assert.Equal(t, "brew upgrade topo", cmd)
	})

	t.Run("topo installed via our script is upgradeable via topo", func(t *testing.T) {
		isManagedByUs, cmd := upgrade.GetUpgradeCommand("/usr/bin/topo")

		assert.True(t, isManagedByUs)
		assert.Equal(t, "topo upgrade", cmd)
	})
}

func TestIsBinaryManagedByHomebrew(t *testing.T) {
	tests := []struct {
		name    string
		binPath string
		want    bool
	}{
		{
			name:    "detects Apple Silicon Homebrew Cellar path",
			binPath: "/opt/homebrew/Cellar/topo/4.1.0/bin/topo",
			want:    true,
		},
		{
			name:    "detects Intel macOS Homebrew Cellar path",
			binPath: "/usr/local/Cellar/topo/4.1.0/bin/topo",
			want:    true,
		},
		{
			name:    "detects Linuxbrew Cellar path",
			binPath: "/home/linuxbrew/.linuxbrew/Cellar/topo/4.1.0/bin/topo",
			want:    true,
		},
		{
			name:    "ignores install script default path",
			binPath: "/Users/alice/.local/bin/topo",
			want:    false,
		},
		{
			name:    "ignores path for another Homebrew formula",
			binPath: "/opt/homebrew/Cellar/other/1.0.0/bin/topo",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := upgrade.IsBinaryManagedByHomebrew(tt.binPath)

			assert.Equal(t, tt.want, got)
		})
	}
}
