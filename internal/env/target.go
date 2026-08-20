package env

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/ssh"
)

const (
	TargetVariable         = "TOPO_TARGET"
	TargetHostnameVariable = "TOPO_TARGET_HOSTNAME"
)

func SetTargetEnvironment(destination ssh.Destination) error {
	variables := map[string]string{
		TargetVariable:         destination.String(),
		TargetHostnameVariable: ssh.NewConfig(destination).HostName,
	}
	for name, value := range variables {
		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", name, err)
		}
	}
	return nil
}
