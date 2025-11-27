package operation

import (
	"fmt"
	"io"
	"strings"
)

type Sequence struct {
	operations []Operation
}

func NewSequence(operations ...Operation) Sequence {
	return Sequence{
		operations: operations,
	}
}

func (s Sequence) Description() string {
	return ""
}

func (s Sequence) Run(cmdOutput io.Writer) error {
	for _, op := range s.operations {
		if cmdOutput != nil {
			printHeader(cmdOutput, op.Description())
		}
		if err := op.Run(cmdOutput); err != nil {
			return err
		}
	}
	return nil
}

func (s Sequence) DryRun(output io.Writer) error {
	for _, op := range s.operations {
		printHeader(output, op.Description())
		if err := op.DryRun(output); err != nil {
			return err
		}
	}
	return nil
}

func printHeader(w io.Writer, description string) {
	if description == "" {
		return
	}

	const totalWidth = 60
	prefix := "┌─ "
	suffix := " "

	descriptionWidth := len(description)
	barWidth := max(totalWidth-len(prefix)-descriptionWidth-len(suffix), 0)

	header := prefix + description + suffix + strings.Repeat("─", barWidth)
	fmt.Fprintln(w)
	fmt.Fprintln(w, header)
}
