package upgrade_test

import (
	"testing"

	"github.com/arm/topo/internal/upgrade"
	"github.com/stretchr/testify/assert"
)

func TestIsTopoBinaryManagedExternally(t *testing.T) {
	tests := []struct {
		name     string
		binPath  string
		expected bool
	}{
		{
			name:     "detects Apple Silicon Homebrew Cellar path",
			binPath:  "/opt/homebrew/Cellar/topo/4.1.0/bin/topo",
			expected: true,
		},
		{
			name:     "detects Intel macOS Homebrew Cellar path",
			binPath:  "/usr/local/Cellar/topo/4.1.0/bin/topo",
			expected: true,
		},
		{
			name:     "detects Linuxbrew Cellar path",
			binPath:  "/home/linuxbrew/.linuxbrew/Cellar/topo/4.1.0/bin/topo",
			expected: true,
		},
		{
			name:     "ignores install script default path",
			binPath:  "/Users/alice/.local/bin/topo",
			expected: false,
		},
		{
			name:     "ignores path for another Homebrew formula",
			binPath:  "/opt/homebrew/Cellar/other/1.0.0/bin/topo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := upgrade.IsTopoBinaryManagedExternally(tt.binPath)

			assert.Equal(t, tt.expected, got)
		})
	}
}
