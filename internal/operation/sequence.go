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
