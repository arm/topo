package operation

import (
	"io"

	"github.com/arm/topo/internal/output/term"
)

type Sequence []Operation

func NewSequence(operations ...Operation) Sequence {
	return operations
}

func (s Sequence) Run(cmdOutput io.Writer) error {
	commandOutput := term.NewCommandOutput(cmdOutput)
	for _, op := range s {
		output := cmdOutput
		if cmdOutput != nil {
			output = term.NewSectionPrinter(commandOutput, op.Description())
		}
		if err := op.Run(output); err != nil {
			return err
		}
	}
	if cmdOutput != nil && len(s) > 0 {
		return commandOutput.Finish()
	}
	return nil
}
