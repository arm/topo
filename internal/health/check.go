package health

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/upgrade"
	"github.com/arm/topo/internal/version"
)

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

func CheckTopoIsUpToDate(ctx context.Context) *CheckFailure {
	if version.Version == version.Dev {
		return nil
	}

	binPath, binPathErr := upgrade.CurrentBinaryPath()

	var latest string
	var err error
	if binPathErr == nil && upgrade.IsBinaryManagedByHomebrew(binPath) {
		latest, err = version.FetchLatestHomebrew(ctx, version.HomebrewFormulaURL)
	} else {
		latest, err = version.FetchLatestArtifactory(ctx, version.ArtifactoryBaseURL)
	}
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to fetch latest version: %v", err))
		return nil
	}
	if latest == version.Version {
		return nil
	}

	fix := Fix{Description: "Upgrade Topo"}
	if binPathErr == nil {
		_, fix.Command = upgrade.GetUpgradeCommand(binPath)
	}
	return &CheckFailure{
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("out of date - current: %s, latest version: %s", version.Version, latest),
		Fix:      &fix,
	}
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
