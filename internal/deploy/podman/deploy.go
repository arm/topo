package podman

import (
	"context"
	"io"

	"github.com/arm/topo/internal/output/term"
)

func Deploy(ctx context.Context, output io.Writer, composeFile string) error {
	socket, err := ResolveLocalComposeSocket(ctx)
	if err != nil {
		return err
	}
	if err := term.PrintHeader(output, "Build images"); err != nil {
		return err
	}
	if err := BuildImages(ctx, output, socket, composeFile); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Pull images"); err != nil {
		return err
	}
	if err := PullImages(ctx, output, socket, composeFile); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Start services"); err != nil {
		return err
	}
	return StartServices(ctx, output, socket, composeFile)
}
