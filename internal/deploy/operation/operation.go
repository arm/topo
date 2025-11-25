package operation

import "io"

type Operation interface {
	Description() string
	Run() error
	DryRun(io.Writer) error
}
