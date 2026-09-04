package term

import (
	"fmt"
	"io"
	"strings"
)

func PrintHeader(w io.Writer, description string) error {
	header := Header(description, IsTTY(w))
	if header == "" {
		return nil
	}

	_, err := fmt.Fprintf(w, "\n%s\n", header)
	return err
}

func Header(description string, isTTY bool) string {
	if description == "" {
		return ""
	}

	const totalWidth = 60
	prefix := "┌─ "
	suffix := " "

	descriptionWidth := len(description)
	barWidth := max(totalWidth-len(prefix)-descriptionWidth-len(suffix), 0)
	bar := suffix + strings.Repeat("─", barWidth)
	if !isTTY {
		return prefix + description + bar
	}
	return Color(Dim, prefix) + description + Color(Dim, bar)
}
