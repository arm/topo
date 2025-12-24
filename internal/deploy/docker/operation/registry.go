package operation

import (
	"fmt"
	"io"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

const (
	RegistryContainerName = "topo-registry"
	registryImage         = "registry:2"
)

func NewRunRegistry() operation.Sequence {
	return operation.NewSequence(
		NewDockerPull(ssh.PlainLocalhost, registryImage),
		operation.NewConditional(
			NewContainerExistsPredicate(ssh.PlainLocalhost, RegistryContainerName),
			NewDockerStart(ssh.PlainLocalhost, RegistryContainerName),
			NewDockerRun(ssh.PlainLocalhost, registryImage, RegistryContainerName,
				[]string{
					"-d",
					"--restart", "always",
					"-p", fmt.Sprintf("127.0.0.1:%d:5000", ssh.RegistryPort),
				},
			),
		),
	)
}

type ContainerExistsPredicate struct {
	host          ssh.Host
	containerName string
}

func NewContainerExistsPredicate(host ssh.Host, containerName string) *ContainerExistsPredicate {
	return &ContainerExistsPredicate{host: host, containerName: containerName}
}

func (p *ContainerExistsPredicate) Eval() bool {
	cmd := command.Docker(p.host, "inspect", p.containerName)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}
