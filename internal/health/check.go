package health

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/version"
)

type Check interface {
	Run(ctx context.Context, r runner.Runner, dep Dependency) *CheckFailure
}

type CheckFailure struct {
	Severity CheckSeverity
	Message  string
	Fix      *Fix
}

type Fix struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
}

type CheckSeverity int

const (
	SeverityError CheckSeverity = iota
	SeverityWarning
	SeverityInfo
)

type CommandSuccessful struct {
	Cmd string
	Fix *Fix
}

func (c CommandSuccessful) Run(ctx context.Context, r runner.Runner, dep Dependency) *CheckFailure {
	_, _, err := r.Run(ctx, c.Cmd)
	if err != nil {
		return &CheckFailure{Message: err.Error(), Fix: c.Fix}
	}
	return nil
}

type BinaryExists struct {
	Severity CheckSeverity
	Fix      *Fix
}

func (b BinaryExists) Run(ctx context.Context, r runner.Runner, dep Dependency) *CheckFailure {
	if err := r.BinaryExists(ctx, dep.Binary); err != nil {
		return &CheckFailure{Severity: b.Severity, Message: err.Error(), Fix: b.Fix}
	}
	return nil
}

type VersionMatches struct {
	CurrentVersion string
	FetchLatest    func(ctx context.Context) (string, error)
	BuildFix       func() Fix
}

func (v VersionMatches) Run(ctx context.Context, _ runner.Runner, _ Dependency) *CheckFailure {
	latest, err := v.FetchLatest(ctx)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to fetch latest version: %v", err))
		return nil
	}
	if latest == v.CurrentVersion {
		return nil
	}

	fix := Fix{}
	if v.BuildFix != nil {
		fix = v.BuildFix()
	}

	return &CheckFailure{
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("out of date - current: %s, latest version: %s", v.CurrentVersion, latest),
		Fix:      &fix,
	}
}

type OpenSSHAvailable struct{}

func (o OpenSSHAvailable) Run(ctx context.Context, r runner.Runner, dep Dependency) *CheckFailure {
	_, stderr, err := r.Run(ctx, "ssh -V")
	if err != nil {
		return &CheckFailure{Message: err.Error()}
	}
	if !strings.Contains(stderr, "OpenSSH_") {
		return &CheckFailure{
			Message: fmt.Sprintf("%q does not resolve to OpenSSH: %s", dep.Binary, stderr),
			Fix: &Fix{
				Description: "Install OpenSSH and ensure its ssh executable is first on PATH",
			},
		}
	}
	return nil
}

type DockerComposeMinVersion struct {
	MinVersion string
}

func (c DockerComposeMinVersion) Run(ctx context.Context, r runner.Runner, _ Dependency) *CheckFailure {
	stdout, _, err := r.Run(ctx, "docker compose version --format json")
	if err != nil {
		return &CheckFailure{Message: err.Error()}
	}

	var output struct {
		Version string `json:"version"`
	}
	err = json.Unmarshal([]byte(stdout), &output)
	if err != nil {
		return &CheckFailure{Message: err.Error()}
	}

	if !version.IsAtLeastVersion(output.Version, c.MinVersion) {
		return &CheckFailure{
			Message: fmt.Sprintf("installed docker compose version %s is older than required version %s", output.Version, c.MinVersion),
			Fix: &Fix{
				Description: fmt.Sprintf("Upgrade Docker Compose to version %s or later. See %s", c.MinVersion, containerEngineInstallURL),
			},
		}
	}

	return nil
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
