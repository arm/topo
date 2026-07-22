package version_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/version"
	"github.com/stretchr/testify/require"
)

func TestIsAtLeastVersion(t *testing.T) {
	tests := []struct {
		current, minimum string
		expected         bool
	}{
		{"v1.0.0", "v1.0.0", true},
		{"v1.2.3", "v1.0.0", true},
		{"v1.9.9", "v1.10.0", false},
		{"v2.0.0-alpha", "v2.0.0", true},
		{"v2.1.0", "2.0.0", true},
		{"v1.0.0", "invalid", true},
		{"invalid", "v1.0.0", false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("current = %s, minimum = %s", test.current, test.minimum), func(t *testing.T) {
			result := version.IsAtLeastVersion(test.current, test.minimum)

			require.Equal(t, test.expected, result)
		})
	}
}
