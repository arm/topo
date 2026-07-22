package health

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/runner"
)

type Check interface {
	Run(ctx context.Context, r runner.Runner, dep Dependency) (*Fix, error)
}

type Fix struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
}

type CheckSeverity int

const (
	SeverityError CheckSeverity = iota
	SeverityWarning
)

type CommandSuccessful struct {
	Cmd string
	Fix *Fix
}

func (c CommandSuccessful) Run(ctx context.Context, r runner.Runner, dep Dependency) (*Fix, error) {
	_, _, err := r.Run(ctx, c.Cmd)
	return c.Fix, err
}

type BinaryExists struct {
	Severity CheckSeverity
	Fix      *Fix
}

func (b BinaryExists) Run(ctx context.Context, r runner.Runner, dep Dependency) (*Fix, error) {
	if err := r.BinaryExists(ctx, dep.Binary); err != nil {
		if errors.Is(err, runner.ErrTimeout) {
			return nil, err
		}
		if b.Severity == SeverityWarning {
			err = WarningError{Err: err}
		}
		return b.Fix, err
	}
	return nil, nil
}

type VersionMatches struct {
	CurrentVersion string
	FetchLatest    func(ctx context.Context) (string, error)
	BuildFix       func() Fix
}

func (v VersionMatches) Run(ctx context.Context, _ runner.Runner, _ Dependency) (*Fix, error) {
	latest, err := v.FetchLatest(ctx)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to fetch latest version: %v", err))
		return nil, nil
	}
	if latest == v.CurrentVersion {
		return nil, nil
	}

	fix := Fix{}
	if v.BuildFix != nil {
		fix = v.BuildFix()
	}

	return &fix, InfoError{Err: fmt.Errorf("out of date - current: %s, latest version: %s", v.CurrentVersion, latest)}
}

type OpenSSHAvailable struct{}

func (o OpenSSHAvailable) Run(ctx context.Context, r runner.Runner, dep Dependency) (*Fix, error) {
	_, stderr, err := r.Run(ctx, "ssh -V")
	if err != nil {
		return nil, err
	}
	if !strings.Contains(stderr, "OpenSSH_") {
		return &Fix{
			Description: "Install OpenSSH and ensure its ssh executable is first on PATH",
		}, fmt.Errorf("%q does not resolve to OpenSSH: %s", dep.Binary, stderr)
	}
	return nil, nil
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
