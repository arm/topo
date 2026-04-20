package health

import (
	"context"
	"fmt"

	"github.com/arm/topo/internal/output/logger"
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
	err := r.BinaryExists(ctx, dep.Binary)
	if b.Severity == SeverityWarning && err != nil {
		err = WarningError{Err: err}
	}
	return b.Fix, err
}

type VersionMatches struct {
	CurrentVersion string
	FetchLatest    func(ctx context.Context) (string, error)
	Fix            string
}

func (v VersionMatches) Run(ctx context.Context, r runner.Runner, dep Dependency) (string, error) {
	latest, err := v.FetchLatest(ctx)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to fetch latest version: %v", err))
		return "", nil
	}
	if latest == v.CurrentVersion {
		return "", nil
	}

	return v.Fix, fmt.Errorf("out of date - current: %s, latest version: %s", v.CurrentVersion, latest)
}
