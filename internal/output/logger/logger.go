package logger

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/arm/topo/internal/output/term"
	"github.com/lmittmann/tint"
)

type Options struct {
	Output io.Writer
	Format term.Format
}

var logger = new(Options{})

func SetOptions(o Options) {
	logger = new(o)
}

func new(o Options) *slog.Logger {
	if o.Output == nil {
		o.Output = io.Writer(os.Stderr)
	}

	switch o.Format {
	case term.JSON:
		return slog.New(slog.NewJSONHandler(o.Output, nil))
	default:
		return slog.New(tint.NewHandler(o.Output, &tint.Options{
			TimeFormat: time.TimeOnly,
			NoColor:    !term.IsTTY(o.Output),
		}))
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
