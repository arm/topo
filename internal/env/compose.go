package env

import "github.com/arm/topo/internal/ssh"

const (
	target   = "TOPO_TARGET"
	hostName = "TOPO_TARGET_HOSTNAME"
)

func variable(name, value string) string {
	return name + "=" + value
}

func ComposeEnv(targetDestination ssh.Destination) []string {
	return []string{
		variable(target, targetDestination.String()),
		variable(hostName, ssh.NewConfig(targetDestination).HostName),
	}
}
