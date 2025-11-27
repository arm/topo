package operation

import "io"

type Operation interface {
	Description() string
	Run(cmdOutput io.Writer) error
	DryRun(output io.Writer) error
}
