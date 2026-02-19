package checks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/compose"
)

const linuxArm64Platform = "linux/arm64"

func isPlatformMissing(platform string) bool {
	return platform == ""
}

func isPlatformMismatch(platform string) bool {
	return !strings.HasPrefix(platform, linuxArm64Platform)
}

func EnsureProjectIsLinuxArm64Ready(composePath string) error {
	project, err := compose.ReadProject(composePath)
	if err != nil {
		return fmt.Errorf("failed to load compose project: %w", err)
	}

	serviceNames := project.ServiceNames()
	builder := strings.Builder{}

	for _, svcName := range serviceNames {
		svc := project.Services[svcName]

		runtime := strings.ToLower(strings.TrimSpace(svc.Runtime))
		if runtime != "" && strings.Contains(runtime, "remoteproc") {
			continue
		}

		if isPlatformMissing(svc.Platform) {
			builder.WriteString(fmt.Sprintf("- service %q is missing platform declaration (set platform: %s or configure remoteproc)\n", svcName, linuxArm64Platform))
		} else if isPlatformMismatch(svc.Platform) {
			builder.WriteString(fmt.Sprintf("- service %q declares platform %q (expected %s)\n", svcName, svc.Platform, linuxArm64Platform))
		}
	}

	if builder.Len() > 0 {
		return errors.New("project is not ready for linux/arm64 deployments:\n" + builder.String())
	}

	return nil
}
