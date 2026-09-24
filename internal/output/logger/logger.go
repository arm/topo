package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/arm/topo/internal/output/term"
)

type Level int

const (
	LevelInfo Level = iota
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	default:
		return "ERROR"
	}
}

func (l Level) Color() string {
	switch l {
	case LevelInfo:
		return term.Blue
	case LevelWarn:
		return term.Yellow
	default:
		return term.Red
	}
}

type Logger struct {
	output  io.Writer
	format  term.Format
	palette term.Palette
}

type Options struct {
	Output io.Writer
	Format term.Format
}

type jsonEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"msg"`
}

func New(opts Options) *Logger {
	if opts.Output == nil {
		opts.Output = os.Stderr
	}

	return &Logger{
		output:  opts.Output,
		format:  opts.Format,
		palette: term.NewPaletteFor(opts.Output),
	}
}

func (l *Logger) Log(level Level, msg string) {
	formattedMsg := formatMessage(l.format, l.palette, level, msg)
	_, _ = fmt.Fprintln(l.output, formattedMsg)
}

func formatMessage(format term.Format, palette term.Palette, level Level, msg string) string {
	timestamp := time.Now().Format(time.TimeOnly)
	if format == term.JSON {
		entry := jsonEntry{
			Time:    timestamp,
			Level:   level.String(),
			Message: msg,
		}
		formattedMsg, _ := json.Marshal(entry)
		return string(formattedMsg)
	}

	timestampStr := palette.Color(term.Dim, timestamp)
	levelStr := palette.Color(level.Color(), level.String())
	return fmt.Sprintf("%s %s %s", timestampStr, levelStr, msg)
}
