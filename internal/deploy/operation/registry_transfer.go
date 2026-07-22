package operation

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
)

type RegistryTransfer struct {
	composeFile string
	source      command.Host
	host        command.Host
	port        string
}

func NewRegistryTransfer(composeFile string, sourceHost, dest command.Host, port string) *RegistryTransfer {
	return &RegistryTransfer{composeFile: composeFile, source: sourceHost, host: dest, port: port}
}

func (r *RegistryTransfer) Description() string {
	return "Transfer via registry"
}

func (r *RegistryTransfer) Run(output io.Writer) error {
	return docker.TransferImagesViaRegistry(context.Background(), output, r.composeFile, r.source, r.host, r.port)
}
