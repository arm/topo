package env

import (
	"fmt"

	"github.com/arm/topo/internal/ssh"
)

const (
	TargetVariable         = "TOPO_TARGET"
	TargetHostnameVariable = "TOPO_TARGET_HOSTNAME"
)

func ResolveTargetEnv(target string) ([]string, error) {
	destination := ssh.NewDestination(target)
	hostname, err := ssh.ResolveHostname(destination)
	if err != nil {
		return nil, err
	}

	return []string{
		fmt.Sprintf("%s=%s", TargetVariable, destination.String()),
		fmt.Sprintf("%s=%s", TargetHostnameVariable, hostname),
	}, nil
}
