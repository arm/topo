package operation

import "io"

type Sequence []Operation

func (s Sequence) Run() error {
	for _, op := range s {
		if err := op.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (s Sequence) DryRun(w io.Writer) error {
	for _, op := range s {
		if err := op.DryRun(w); err != nil {
			return err
		}
	}
	return nil
}
