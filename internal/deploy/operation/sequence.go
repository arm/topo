package operation

import (
	"fmt"
	"io"
	"strings"
)

type Sequence struct {
	cmdOutput  io.Writer
	operations []Operation
}

func NewSequence(cmdOutput io.Writer, operations ...Operation) Sequence {
	return Sequence{
		cmdOutput:  cmdOutput,
		operations: operations,
	}
}

func (s Sequence) Description() string {
	return ""
}

func (s Sequence) Run() error {
	for _, op := range s.operations {
		if s.cmdOutput != nil {
			printHeader(s.cmdOutput, op.Description())
		}
		if err := op.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (s Sequence) DryRun(w io.Writer) error {
	for _, op := range s.operations {
		printHeader(w, op.Description())
		if err := op.DryRun(w); err != nil {
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
