package output

import (
	"fmt"
	"io"
)

// Output format for the commands
type Format int

const (
	// PlainFormat renders human-readable plain text
	PlainFormat Format = iota
	// JSONFormat renders machine-readable JSON
	JSONFormat
)

type Printable interface {
	AsJSON() (string, error)
	AsPlain() (string, error)
}

type Printer struct {
	target io.Writer
	format Format
}

func NewPrinter(target io.Writer, format Format) *Printer {
	return &Printer{target: target, format: format}
}

func (p *Printer) Print(printable Printable) error {
	if p.format == JSONFormat {
		jsonStr, err := printable.AsJSON()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(p.target, jsonStr)
		return err
	}

	plainStr, err := printable.AsPlain()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.target, plainStr)
	return err
}
