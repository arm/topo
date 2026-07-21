package command_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapInLoginShell(t *testing.T) {
	t.Run("wraps command in login shell", func(t *testing.T) {
		got := command.WrapInLoginShell("echo $PATH")

		want := `/bin/sh -c "exec 3>&1 4>&2; exec ${SHELL:-/bin/sh} -l -c \"exec 1>&3 2>&4 3>&- 4>&-; echo \\\$PATH\" >/dev/null 2>&1"`
		assert.Equal(t, want, got)
	})

	t.Run("keeps login shell output separate from command output", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("test uses a POSIX shell script")
		}

		fakeShell := filepath.Join(t.TempDir(), "fake-shell")
		err := os.WriteFile(fakeShell, []byte(`#!/bin/sh
printf 'login shell stdout\n'
printf 'login shell stderr\n' >&2
while [ "$#" -gt 0 ]; do
	if [ "$1" = "-c" ]; then
		exec /bin/sh -c "$2"
	fi
	shift
done
exit 1
`), 0o700)
		require.NoError(t, err)

		wrapped := command.WrapInLoginShell("printf 'command output\\n'; printf 'command error\\n' >&2")
		cmd := exec.Command("/bin/sh", "-c", wrapped)
		cmd.Env = append(os.Environ(), "SHELL="+fakeShell)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()

		require.NoError(t, err)
		assert.Equal(t, "command output\n", stdout.String())
		assert.Equal(t, "command error\n", stderr.String())
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
