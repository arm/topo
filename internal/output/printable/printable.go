package printable

import (
	"fmt"
	"io"

	"github.com/arm/topo/internal/output/term"
)

type Printable interface {
	AsJSON() (string, error)
	AsPlain(isTTY bool) (string, error)
}

func Print(p Printable, w io.Writer, f term.Format) error {
	var out string
	var err error

	if f == term.JSON {
		out, err = p.AsJSON()
		if err != nil {
			return fmt.Errorf("render printable as JSON: %w", err)
		}
	} else {
		out, err = p.AsPlain(term.IsTTY(w))
		if err != nil {
			return fmt.Errorf("render printable as plain text: %w", err)
		}
	}

	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("write printable output: %w", err)
	}
	return nil
}
