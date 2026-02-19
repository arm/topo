package logger

import (
	"github.com/arm/topo/internal/output/term"
)

type (
	Level string
)

const (
	Info    Level = "INFO"
	Warning Level = "WARN"
	Err     Level = "ERROR"
)

func (l Level) Color() string {
	switch l {
	case Warning:
		return term.Yellow
	case Err:
		return term.Red
	default:
		return term.Reset
	}
}

type Logger interface {
	Log(e ...Entry)
}

type Entry struct {
	Level   Level  `json:"level"`
	Message string `json:"message"`
}
