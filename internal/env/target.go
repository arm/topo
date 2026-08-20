package env

import (
	"github.com/arm/topo/internal/ssh"
)

const (
	TargetVariable         = "TOPO_TARGET"
	TargetHostnameVariable = "TOPO_TARGET_HOSTNAME"
)

func TargetEnv(target string) map[string]string {
	destination := ssh.NewDestination(target)
	return map[string]string{
		TargetVariable:         destination.String(),
		TargetHostnameVariable: ssh.NewConfig(destination).HostName,
	}
}
