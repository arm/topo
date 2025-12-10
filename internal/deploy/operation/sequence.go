package operation

import (
	"fmt"
	"io"
	"strings"
)

type Sequence []Operation

func NewSequence(operations ...Operation) Sequence {
	return operations
}

func (s Sequence) Run(cmdOutput io.Writer) error {
	for _, op := range s {
		if cmdOutput != nil {
			err := printHeader(cmdOutput, op.Description())
			if err != nil {
				return err
			}
		}
		if err := op.Run(cmdOutput); err != nil {
			return err
		}
	}
	return nil
}

func (s Sequence) DryRun(output io.Writer) error {
	for _, op := range s {
		err := printHeader(output, op.Description())
		if err != nil {
			return err
		}
		if err := op.DryRun(output); err != nil {
			return err
		}
	}
	return nil
}

func printHeader(w io.Writer, description string) error {
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
