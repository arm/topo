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
	printHeader := term.PrintFirstHeader
	for _, op := range s {
		if cmdOutput != nil {
			description := op.Description()
			err := printHeader(cmdOutput, description)
			if err != nil {
				return err
			}
			if description != "" {
				printHeader = term.PrintNthHeader
			}
		}
		if err := op.Run(cmdOutput); err != nil {
			return err
		}
	}
	return nil
}
