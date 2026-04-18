package command

import "github.com/arm/topo/internal/ssh"

type Host struct {
	value string
}

func NewHostFromDestination(dest ssh.Destination) Host {
	if dest.IsPlainLocalhost() {
		return LocalHost
	}
	return Host{value: dest.String()}
}

var LocalHost = Host{value: ""}
