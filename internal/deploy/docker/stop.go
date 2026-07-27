package docker

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
)

func Stop(ctx context.Context, output io.Writer, composeFile string, destination ssh.Destination) error {
	if err := term.PrintHeader(output, "Stop services"); err != nil {
		return err
	}
	return StopServices(ctx, output, composeFile, command.NewHostFromDestination(destination))
}
