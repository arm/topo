package logger

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/arm/topo/internal/output/term"
	"github.com/lmittmann/tint"
)

var (
	output = io.Writer(os.Stderr)
	logger = newPlainLogger()
)

func newPlainLogger() *slog.Logger {
	return slog.New(tint.NewHandler(output, &tint.Options{
		TimeFormat: time.TimeOnly,
		NoColor:    !term.IsTTY(output),
	}))
}

func SetOutput(w io.Writer) {
	output = w
	logger = newPlainLogger()
}

func SetOutputFormat(format term.Format) {
	switch format {
	case term.Plain:
		logger = newPlainLogger()
	case term.JSON:
		logger = slog.New(slog.NewJSONHandler(output, nil))
	}
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}
