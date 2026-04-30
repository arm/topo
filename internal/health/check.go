package health

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/runner"
)

type Check interface {
	Run(ctx context.Context, r runner.Runner, dep Dependency) (string, error)
}

type fixCommandProvider interface {
	fixCommand() string
}

type CheckSeverity int

const (
	SeverityError CheckSeverity = iota
	SeverityWarning
)

type CommandSuccessful struct {
	Cmd        string
	Fix        string
	FixCommand string
}

func (c CommandSuccessful) Run(ctx context.Context, r runner.Runner, dep Dependency) (string, error) {
	_, err := r.Run(ctx, c.Cmd)
	return c.Fix, err
}

func (c CommandSuccessful) fixCommand() string {
	return c.FixCommand
}

type BinaryExists struct {
	Severity   CheckSeverity
	Fix        string
	FixCommand string
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

func (b BinaryExists) fixCommand() string {
	return b.FixCommand
}

type VersionMatches struct {
	CurrentVersion string
	FetchLatest    func(ctx context.Context) (string, error)
	Fix            string
	FixCommand     string
}

func (v VersionMatches) Run(ctx context.Context, _ runner.Runner, _ Dependency) (string, error) {
	latest, err := v.FetchLatest(ctx)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to fetch latest version: %v", err))
		return "", nil
	}
	if latest == v.CurrentVersion {
		return "", nil
	}

	return v.Fix, InfoError{Err: fmt.Errorf("out of date - current: %s, latest version: %s", v.CurrentVersion, latest)}
}

func (v VersionMatches) fixCommand() string {
	return v.FixCommand
}

func RemoveVersionChecks(deps []Dependency) []Dependency {
	deps = slices.Clone(deps)
	for i, dep := range deps {
		deps[i].Checks = slices.DeleteFunc(dep.Checks, func(c Check) bool {
			_, ok := c.(VersionMatches)
			return ok
		})
	}
	return deps
}
