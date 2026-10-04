package deploy

import "github.com/arm/topo/internal/ssh"

const (
	DefaultRegistryContainerName = "topo-registry"
	DefaultRegistryPort          = "12737"
)

type RegistryConfig struct {
	ContainerName       string
	Port                string
	SkipRemotePortCheck bool
}

func (config RegistryConfig) WithDefaults() RegistryConfig {
	if config.ContainerName == "" {
		config.ContainerName = DefaultRegistryContainerName
	}
	if config.Port == "" {
		config.Port = DefaultRegistryPort
	}
	return config
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
