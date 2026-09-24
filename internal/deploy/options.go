package deploy

import "github.com/arm/topo/internal/ssh"

type RegistryConfig struct {
	ContainerName       string
	Port                string
	SkipRemotePortCheck bool
}

type RecreateMode int

const (
	RecreateModeDefault RecreateMode = iota
	RecreateModeForce
	RecreateModeNone
)

type Options struct {
	RecreateMode          RecreateMode
	TargetHost            ssh.Destination
	Registry              *RegistryConfig
	DefaultSuccessMessage string
}
