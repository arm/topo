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

func SetTargetEnv(target string) error {
	destination := ssh.NewDestination(target)
	hostname, err := ssh.ResolveHostname(destination)
	if err != nil {
		return err
	}
	vars := map[string]string{
		TargetVariable:         destination.String(),
		TargetHostnameVariable: hostname,
	}
	for name, value := range vars {
		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", name, err)
		}
	}
	return nil
}
