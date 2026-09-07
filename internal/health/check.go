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
	Run(ctx context.Context, r runner.Runner) *CheckFailure
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

func (c CommandSuccessful) Run(ctx context.Context, r runner.Runner) *CheckFailure {
	return CheckCommandSuccessful(ctx, r, c.Cmd, c.Fix)
}

func CheckCommandSuccessful(ctx context.Context, r runner.Runner, command string, fix *Fix) *CheckFailure {
	_, _, err := r.Run(ctx, command)
	if err != nil {
		return &CheckFailure{Message: err.Error(), Fix: fix}
	}
	return nil
}

type BinaryExists struct {
	Binary   string
	Severity CheckSeverity
	Fix      *Fix
}

func (b BinaryExists) Run(ctx context.Context, r runner.Runner) *CheckFailure {
	return CheckBinaryExists(ctx, r, b.Binary, b.Severity, b.Fix)
}

func CheckBinaryExists(ctx context.Context, r runner.Runner, binary string, severity CheckSeverity, fix *Fix) *CheckFailure {
	if err := r.BinaryExists(ctx, binary); err != nil {
		return &CheckFailure{Severity: severity, Message: err.Error(), Fix: fix}
	}
	return nil
}

type VersionMatches struct {
	CurrentVersion string
	FetchLatest    func(ctx context.Context) (string, error)
	BuildFix       func() Fix
}

func (v VersionMatches) Run(ctx context.Context, _ runner.Runner) *CheckFailure {
	return CheckVersionMatches(ctx, v.CurrentVersion, v.FetchLatest, v.BuildFix)
}

func CheckVersionMatches(ctx context.Context, currentVersion string, fetchLatest func(context.Context) (string, error), buildFix func() Fix) *CheckFailure {
	latest, err := fetchLatest(ctx)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to fetch latest version: %v", err))
		return nil
	}
	if latest == currentVersion {
		return nil
	}

	fix := Fix{}
	if buildFix != nil {
		fix = buildFix()
	}

	return &CheckFailure{
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("out of date - current: %s, latest version: %s", currentVersion, latest),
		Fix:      &fix,
	}
}

type OpenSSHAvailable struct {
	SSHBinary string
}

func (o OpenSSHAvailable) Run(ctx context.Context, r runner.Runner) *CheckFailure {
	return CheckOpenSSHAvailable(ctx, r, o.SSHBinary)
}

func CheckOpenSSHAvailable(ctx context.Context, r runner.Runner, sshBinary string) *CheckFailure {
	_, stderr, err := r.Run(ctx, sshBinary+" -V")
	if err != nil {
		return &CheckFailure{Message: err.Error()}
	}
	if !strings.Contains(stderr, "OpenSSH_") {
		return &CheckFailure{
			Message: fmt.Sprintf("%q does not resolve to OpenSSH: %s", sshBinary, stderr),
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

func (c DockerComposeMinVersion) Run(ctx context.Context, r runner.Runner) *CheckFailure {
	return CheckDockerComposeMinVersion(ctx, r, c.MinVersion)
}

func CheckDockerComposeMinVersion(ctx context.Context, r runner.Runner, minVersion string) *CheckFailure {
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

	if !version.IsAtLeastVersion(output.Version, minVersion) {
		return &CheckFailure{
			Message: fmt.Sprintf("installed docker compose version %s is older than required version %s", output.Version, minVersion),
			Fix: &Fix{
				Description: fmt.Sprintf("Upgrade Docker Compose to version %s or later. See %s", minVersion, containerEngineInstallURL),
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
