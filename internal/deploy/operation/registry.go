package operation

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/operation"
)

func NewRunRegistry(containerName, port string) operation.Sequence {
	return operation.NewSequence(&runRegistry{containerName: containerName, port: port})
}

type runRegistry struct {
	containerName string
	port          string
}

func (r *runRegistry) Description() string {
	return "Run registry"
}

func (r *runRegistry) Run(output io.Writer) error {
	return docker.EnsureRegistryRunning(context.Background(), output, r.containerName, r.port)
}
