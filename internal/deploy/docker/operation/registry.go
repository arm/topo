package operation

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/deploy/operation"
	"github.com/arm/topo/internal/ssh"
)

const (
	RegistryContainerName = "topo-registry"
	DefaultRegistryPort   = "12737"
	registryImage         = "registry:2"
)

func NewRunRegistry(port string) operation.Sequence {
	return operation.NewSequence(
		NewDockerPull(ssh.PlainLocalhost, registryImage),
		operation.NewConditional(
			NewContainerExistsPredicate(ssh.PlainLocalhost, RegistryContainerName),
			NewDockerStart(ssh.PlainLocalhost, RegistryContainerName),
			NewRegistryRunWrapper(NewDockerRun(ssh.PlainLocalhost, registryImage, RegistryContainerName,
				[]string{
					"-d",
					"--restart", "always",
					"-p", fmt.Sprintf("127.0.0.1:%s:5000", port),
				},
			)),
		),
	)
}

type RegistryRunWrapper struct {
	*Docker
}

func NewRegistryRunWrapper(d *Docker) *RegistryRunWrapper {
	return &RegistryRunWrapper{Docker: d}
}

func (r *RegistryRunWrapper) Run(w io.Writer) error {
	var buf bytes.Buffer
	combined := io.MultiWriter(w, &buf)
	if err := r.Docker.Run(combined); err != nil {
		if strings.Contains(buf.String(), "already in use") || strings.Contains(buf.String(), "already allocated") {
			return fmt.Errorf("%w\nport is already in use, this could be an existing %s or another process", err, RegistryContainerName)
		}
		return err
	}
	return nil
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
