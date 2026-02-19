package console_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/arm/topo/internal/output/console"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
)

func TestConsoleLoggerLog(t *testing.T) {
	t.Run("writes plain log to stderr without color", func(t *testing.T) {
		var errBuf bytes.Buffer

		l := console.NewLogger(
			&errBuf,
			term.Plain,
		)

		entry := logger.Entry{
			Level:   logger.Err,
			Message: "something happened",
		}

		l.Log(entry)

		assert.Equal(t, "ERROR: something happened\n", errBuf.String())
	})

	t.Run("writes JSON log to stderr", func(t *testing.T) {
		var errBuf bytes.Buffer

		l := console.NewLogger(
			&errBuf,
			term.JSON,
		)

		entry := logger.Entry{
			Level:   logger.Err,
			Message: "uh oh",
		}

		l.Log(entry)

		var decoded logger.Entry
		err := json.Unmarshal(errBuf.Bytes(), &decoded)

		assert.NoError(t, err)
		assert.Equal(t, entry.Level, decoded.Level)
		assert.Equal(t, entry.Message, decoded.Message)
	})
}
