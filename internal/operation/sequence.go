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
	progress := term.NewProgress(cmdOutput)
	for _, op := range s {
		if cmdOutput != nil {
			if err := progress.Header(op.Description()); err != nil {
				return err
			}
		}
		if err := op.Run(cmdOutput); err != nil {
			return err
		}
	}
	return nil
}
