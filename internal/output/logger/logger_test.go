package logger_test

import (
	"bytes"
	"io"
	"runtime"
	"testing"
	"testing/synctest"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	t.Run("logs plain output", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var buf bytes.Buffer
			l := logger.New(logger.Options{Output: &buf})
			want := `00:00:00 INFO Hello World
00:00:00 WARN This is a warning
00:00:00 ERROR This is an error
`

			l.Log(logger.LevelInfo, "Hello World")
			l.Log(logger.LevelWarn, "This is a warning")
			l.Log(logger.LevelError, "This is an error")

			assert.Equal(t, want, buf.String())
		})
	})

	t.Run("clears an unfinished terminal line before logging a warning", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("PTYs are not supported on Windows")
		}
		master, terminal, err := pty.Open()
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, master.Close()) })
		t.Cleanup(func() { assert.NoError(t, terminal.Close()) })
		log := logger.New(logger.Options{Output: terminal})
		want := "Checking health...\r\033[K"
		_, err = terminal.WriteString("Checking health...")
		require.NoError(t, err)

		log.Log(logger.LevelWarn, "Continuing with Podman checks.")

		got := make([]byte, len(want))
		_, err = io.ReadFull(master, got)
		require.NoError(t, err)
		assert.Equal(t, want, string(got))
	})

	t.Run("logs JSON output", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var buf bytes.Buffer
			l := logger.New(logger.Options{Output: &buf, Format: term.JSON})
			want := `{"time":"00:00:00","level":"INFO","msg":"Hello World"}
{"time":"00:00:00","level":"WARN","msg":"This is a warning"}
{"time":"00:00:00","level":"ERROR","msg":"This is an error"}
`

			l.Log(logger.LevelInfo, "Hello World")
			l.Log(logger.LevelWarn, "This is a warning")
			l.Log(logger.LevelError, "This is an error")

			assert.Equal(t, want, buf.String())
		})
	})
}
