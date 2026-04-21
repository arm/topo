package health

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/runner"
)

type Check interface {
	Run(ctx context.Context, r runner.Runner, dep Dependency) (string, error)
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

func (c CommandSuccessful) Run(ctx context.Context, r runner.Runner, dep Dependency) (string, error) {
	_, err := r.Run(ctx, c.Cmd)
	return c.Fix, err
}

type BinaryExists struct {
	Severity CheckSeverity
	Fix      string
}

func (b BinaryExists) Run(ctx context.Context, r runner.Runner, dep Dependency) (string, error) {
	if err := r.BinaryExists(ctx, dep.Binary); err != nil {
		if errors.Is(err, runner.ErrTimeout) {
			return "", err
		}
		if b.Severity == SeverityWarning {
			err = WarningError{Err: err}
		}
		return b.Fix, err
	}
	return "", nil
}
