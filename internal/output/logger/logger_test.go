package logger_test

import (
	"bytes"
	"testing"
	"testing/synctest"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
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
