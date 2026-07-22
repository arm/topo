package colors

import "fmt"

const reset = "\x1b[0m"

type Role uint8

const (
	Accent Role = iota + 1
	Success
	Warning
	Failure
	Information
	Muted
)

type Palette struct {
	enabled bool
}

func NewPalette(enabled bool) Palette {
	return Palette{enabled: enabled}
}

func (p Palette) Apply(role Role, text string) string {
	code, ok := ansiCodes[role]
	if !ok {
		panic(fmt.Sprintf("unknown color role: %d", role))
	}
	if !p.enabled || text == "" {
		return text
	}

	return code + text + reset
}

var ansiCodes = map[Role]string{
	Accent:      "\x1b[36m",
	Success:     "\x1b[32m",
	Warning:     "\x1b[33m",
	Failure:     "\x1b[31m",
	Information: "\x1b[34m",
	Muted:       "\x1b[2m",
}
