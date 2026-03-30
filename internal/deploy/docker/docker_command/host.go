package dockercommand

import "github.com/arm/topo/internal/ssh"

type Host struct {
	value string
}

func NewHostFromDestination(dest ssh.Destination) Host {
	if dest.IsPlainLocalhost() {
		return NewLocalHost()
	}
	return Host{value: dest.String()}
}

func NewLocalHost() Host {
	return Host{value: ""}
}
