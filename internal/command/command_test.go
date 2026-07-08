package command_test

import (
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapInLoginShell(t *testing.T) {
	t.Run("wraps command in login shell", func(t *testing.T) {
		got := command.WrapInLoginShell("echo $PATH")

		want := `/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"echo \\\$PATH\""`
		assert.Equal(t, want, got)
	})
}

func TestBinaryLookupCommand(t *testing.T) {
	t.Run("returns wrapped command for valid binary", func(t *testing.T) {
		got, err := command.BinaryLookupCommand("docker")

		require.NoError(t, err)
		assert.Equal(t, "command -v docker", got)
	})

	t.Run("returns error for invalid binary", func(t *testing.T) {
		got, err := command.BinaryLookupCommand("bad name")

		assert.Error(t, err)
		assert.Empty(t, got)
	})
}

func TestQuoteArg(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "plain arg",
			arg:  "compose.yml",
			want: "compose.yml",
		},
		{
			name: "empty arg",
			arg:  "",
			want: "''",
		},
		{
			name: "arg with spaces",
			arg:  "custom compose.yaml",
			want: "'custom compose.yaml'",
		},
		{
			name: "arg with apostrophe",
			arg:  "custom's-compose.yaml",
			want: `'custom'"'"'s-compose.yaml'`,
		},
		{
			name: "windows path",
			arg:  `C:\Users\runner\AppData\Local\Temp\custom-compose.yaml`,
			want: `'C:\Users\runner\AppData\Local\Temp\custom-compose.yaml'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := command.QuoteArg(tt.arg)

			assert.Equal(t, tt.want, got)
		})
	}
}
