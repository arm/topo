package docker

import (
	"context"
	"errors"
	"io"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func Stop(ctx context.Context, output io.Writer, scope project.Scope, destination ssh.Destination) error {
	commandOutput := term.NewCommandOutput(output)
	section := term.NewSectionPrinter(commandOutput, "Stop services")
	stopErr := StopServices(ctx, section, NewHostFromDestination(destination), scope)
	return errors.Join(stopErr, commandOutput.Finish())
}
