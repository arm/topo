package operation

import "io"

type Operation interface {
	Run() error
	DryRun(io.Writer) error
}
