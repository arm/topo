package term

import (
	"io"
	"os"
)

const (
	Reset  = "\033[0m"
	Dim    = "\033[2;90m"
	Gray   = "\033[90m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Red    = "\033[31m"
)

type Palette struct {
	enabled bool
}

func NewPalette(enabled bool) Palette {
	return Palette{enabled: enabled}
}

func NewPaletteFor(w io.Writer) Palette {
	allowColor := os.Getenv("NO_COLOR") == ""
	return NewPalette(allowColor && IsTTY(w))
}

func (p Palette) Color(code, text string) string {
	if !p.enabled || text == "" {
		return text
	}

	return code + text + Reset
}
