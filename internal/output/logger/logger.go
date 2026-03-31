package logger

import (
	"log/slog"
	"os"

	"github.com/arm/topo/internal/output/term"
)

func SetOutputFormat(format term.Format) {
	switch format {
	case term.Plain:
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	case term.JSON:
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	}
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}
