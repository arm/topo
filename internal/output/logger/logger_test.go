package logger_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	logger.SetOutputFormat(term.JSON)
	t.Cleanup(func() {
		logger.SetOutput(os.Stderr)
		logger.SetOutputFormat(term.Plain)
	})
	return &buf
}

func TestLogFunctions(t *testing.T) {
	buf := setup(t)

	tests := []struct {
		name  string
		fn    func(string, ...any)
		level string
	}{
		{"Info", logger.Info, "INFO"},
		{"Warn", logger.Warn, "WARN"},
		{"Error", logger.Error, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			tt.fn("hello", "key", "val")

			var entry map[string]any
			err := json.Unmarshal(buf.Bytes(), &entry)
			assert.NoError(t, err)
			assert.Equal(t, tt.level, entry["level"])
			assert.Equal(t, "hello", entry["msg"])
			assert.Equal(t, "val", entry["key"])
		})
	}
}

func TestSetOutputFormat(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		buf := setup(t)

		logger.Info("json test")

		var entry map[string]any
		err := json.Unmarshal(buf.Bytes(), &entry)
		assert.NoError(t, err)
		assert.Equal(t, "INFO", entry["level"])
		assert.Equal(t, "json test", entry["msg"])
	})

	t.Run("Plain", func(t *testing.T) {
		buf := setup(t)
		logger.SetOutputFormat(term.Plain)

		logger.Info("plain test")

		assert.Contains(t, buf.String(), "INF plain test\n")
	})
}
