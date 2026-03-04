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
	for _, op := range s {
		if cmdOutput != nil {
			err := term.PrintHeader(cmdOutput, op.Description())
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
		err := term.PrintHeader(output, op.Description())
		if err != nil {
			return err
		}
		if err := op.DryRun(output); err != nil {
			return err
		}
	}
	return nil
}
