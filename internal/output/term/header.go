package term

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

func PrintFirstHeader(w io.Writer, description string) error {
	return printHeader(w, description, "")
}

func PrintNthHeader(w io.Writer, description string) error {
	return printHeader(w, description, "\n")
}

func printHeader(w io.Writer, description string, prefix string) error {
	header := Header(description, IsTTY(w))
	if header == "" {
		return nil
	}

	_, err := fmt.Fprintf(w, "%s%s\n", prefix, header)
	return err
}

func Header(description string, isTTY bool) string {
	if description == "" {
		return ""
	}

	const totalWidth = 60
	prefix := "── "
	suffix := " "

	descriptionWidth := utf8.RuneCountInString(description)
	barWidth := max(totalWidth-utf8.RuneCountInString(prefix)-descriptionWidth-utf8.RuneCountInString(suffix), 0)
	bar := suffix + strings.Repeat("─", barWidth)
	if !isTTY {
		return prefix + description + bar
	}
	return Color(Dim, prefix) + description + Color(Dim, bar)
}
