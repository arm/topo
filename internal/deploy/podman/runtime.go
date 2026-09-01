package podman

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/compose"
)

func EnsureNoRuntimeSet(composePath string) error {
	project, err := compose.ReadProject(composePath)
	if err != nil {
		return fmt.Errorf("failed to load compose project: %w", err)
	}

	var serviceRuntimes []string
	for _, serviceName := range project.ServiceNames() {
		runtime := strings.TrimSpace(project.Services[serviceName].Runtime)
		if runtime != "" {
			serviceRuntimes = append(serviceRuntimes, fmt.Sprintf("%q service uses %q", serviceName, runtime))
		}
	}
	if len(serviceRuntimes) == 0 {
		return nil
	}

	return errors.New(`specifying "runtime:" in Compose files is unsupported for Podman deployments: ` + strings.Join(serviceRuntimes, ", "))
}
