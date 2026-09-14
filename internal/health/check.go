package health

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/version"
)

func CheckOpenSSHAvailable(ctx context.Context, r runner.Runner, sshBinary string) *DependencyCheckFailure {
	_, stderr, err := r.Run(ctx, sshBinary+" -V")
	if err != nil {
		return &DependencyCheckFailure{Message: err.Error()}
	}
	if !strings.Contains(stderr, "OpenSSH_") {
		return &DependencyCheckFailure{
			Message: fmt.Sprintf("%q does not resolve to OpenSSH: %s", sshBinary, stderr),
			Fix: &Fix{
				Description: "Install OpenSSH and ensure its ssh executable is first on PATH",
			},
		}
	}
	return nil
}

func CheckDockerComposeMinVersion(ctx context.Context, r runner.Runner, minVersion string) *DependencyCheckFailure {
	stdout, _, err := r.Run(ctx, "docker compose version --format json")
	if err != nil {
		return &DependencyCheckFailure{Message: err.Error()}
	}

	var output struct {
		Version string `json:"version"`
	}
	err = json.Unmarshal([]byte(stdout), &output)
	if err != nil {
		return &DependencyCheckFailure{Message: err.Error()}
	}

	if !version.IsAtLeastVersion(output.Version, minVersion) {
		return &DependencyCheckFailure{
			Message: fmt.Sprintf("installed docker compose version %s is older than required version %s", output.Version, minVersion),
			Fix: &Fix{
				Description: fmt.Sprintf("Upgrade Docker Compose to version %s or later. See %s", minVersion, containerEngineInstallURL),
			},
		}
	}

	return nil
}
