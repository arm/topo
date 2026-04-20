package health

import (
	"context"
	"fmt"
	"runtime"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/version"
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

type IsTopoUpToDate struct{}

func (c IsTopoUpToDate) Run(ctx context.Context, r runner.Runner, dep Dependency) (string, error) {
	latest, err := version.FetchLatest(ctx, version.ArtifactoryBaseURL)
	if err != nil {
		logger.Warn("failed to fetch latest topo version: %v", err)
		return "", nil
	}
	if latest == version.Version {
		return "", nil
	}

	fix := "run `curl -fsSL https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.sh | sh`"
	if runtime.GOOS == "windows" {
		fix = "run `irm https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.ps1 | iex`"
	}

	return fix, fmt.Errorf("topo is not up to date - current: %s, latest version: %s", version.Version, latest)
}
