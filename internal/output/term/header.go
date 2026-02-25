package term

import (
	"fmt"
	"io"
	"strings"
)

func PrintHeader(w io.Writer, description string) error {
	if description == "" {
		return nil
	}

	const totalWidth = 60
	prefix := "┌─ "
	suffix := " "

	descriptionWidth := len(description)
	barWidth := max(totalWidth-len(prefix)-descriptionWidth-len(suffix), 0)

	header := prefix + description + suffix + strings.Repeat("─", barWidth)
	_, err := fmt.Fprintf(w, "\n%s\n", header)
	return err
}
