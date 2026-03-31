package logger_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
)

func setupTestLogOutputBuffer(t *testing.T, format term.Format) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	logger.SetOptions(logger.Options{Output: &buf, Format: format})
	t.Cleanup(func() {
		logger.SetOptions(logger.Options{})
	})
	return &buf
}

func TestLogFunctions(t *testing.T) {
	buf := setupTestLogOutputBuffer(t, term.JSON)

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
		buf := setupTestLogOutputBuffer(t, term.JSON)

		logger.Info("json test")

		var entry map[string]any
		err := json.Unmarshal(buf.Bytes(), &entry)
		assert.NoError(t, err)
		assert.Equal(t, "INFO", entry["level"])
		assert.Equal(t, "json test", entry["msg"])
	})

	t.Run("Plain", func(t *testing.T) {
		buf := setupTestLogOutputBuffer(t, term.Plain)

		logger.Info("plain test")

		assert.Contains(t, buf.String(), "INF plain test\n")
	})
}
