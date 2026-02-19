package console

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
)

type Logger struct {
	err         io.Writer
	enableColor bool
	logFormat   term.Format
}

func NewLogger(stderr io.Writer, lf term.Format) *Logger {
	return &Logger{
		err:         stderr,
		enableColor: term.IsTTY(stderr),
		logFormat:   lf,
	}
}

func (c *Logger) Log(entries ...logger.Entry) {
	for _, e := range entries {
		var output string
		if c.logFormat == term.JSON {
			b, _ := json.Marshal(e)
			output = string(b)
		} else {
			output = fmt.Sprintf("%s: %s", e.Level, e.Message)
			if c.enableColor {
				output = term.Color(e.Level.Color(), output)
			}
		}
		_, _ = fmt.Fprintln(c.err, output)
	}
}
