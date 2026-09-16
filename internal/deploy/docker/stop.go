package docker

import (
	"context"
	"io"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func Stop(ctx context.Context, output io.Writer, scope project.Scope, destination ssh.Destination) error {
	if err := term.PrintFirstHeader(output, "Stop services"); err != nil {
		return err
	}
	return StopServices(ctx, output, NewHostFromDestination(destination), scope)
}
