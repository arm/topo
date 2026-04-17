package health

import (
	"context"

	"github.com/arm/topo/internal/runner"
)

type Check interface {
	Apply(ctx context.Context, r runner.Runner, dep Dependency) (string, error)
}

type CheckSeverity int

const (
	SeverityError CheckSeverity = iota
	SeverityWarning
)

type CommandSuccessful struct {
	Cmd string
	Fix string
}

func (c CommandSuccessful) Apply(ctx context.Context, r runner.Runner, dep Dependency) (string, error) {
	_, err := r.Run(ctx, c.Cmd)
	return c.Fix, err
}

type BinaryExists struct {
	Severity CheckSeverity
	Fix      string
}

func (b BinaryExists) Apply(ctx context.Context, r runner.Runner, dep Dependency) (string, error) {
	err := r.BinaryExists(ctx, dep.Binary)
	if b.Severity == SeverityWarning && err != nil {
		err = WarningError{Err: err}
	}
	return b.Fix, err
}
