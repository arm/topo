package podman

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/compose"
)

func EnsureNoRuntimeSet(composePath string) error {
	runtimes, err := listCustomServiceRuntimes(composePath)
	if err != nil {
		return err
	}

	if len(runtimes) == 0 {
		return nil
	}

	return errors.New(`specifying "runtime:" in Compose files is unsupported for Podman deployments: ` + formatServiceRuntimes(runtimes))
}

type serviceRuntime struct {
	serviceName string
	runtime     string
}

func listCustomServiceRuntimes(composePath string) ([]serviceRuntime, error) {
	project, err := compose.ReadProject(composePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load compose project: %w", err)
	}

	var collected []serviceRuntime
	for _, serviceName := range project.ServiceNames() {
		runtime := strings.TrimSpace(project.Services[serviceName].Runtime)
		if runtime != "" {
			collected = append(collected, serviceRuntime{
				serviceName: serviceName,
				runtime:     runtime,
			})
		}
	}

	return collected, nil
}

func formatServiceRuntimes(runtimes []serviceRuntime) string {
	formatted := make([]string, 0, len(runtimes))
	for _, serviceRuntime := range runtimes {
		formatted = append(formatted, fmt.Sprintf("%q service uses %q", serviceRuntime.serviceName, serviceRuntime.runtime))
	}
	return strings.Join(formatted, ", ")
}
