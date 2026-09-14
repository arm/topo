package podman

import (
	"context"
	"errors"
	"io"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func Stop(ctx context.Context, output io.Writer, scope project.Scope, target ssh.Destination) (stopErr error) {
	commandOutput := term.NewCommandOutput(output)
	socket := LocalSocket
	if !target.IsPlainLocalhost() {
		section := term.NewSectionPrinter(commandOutput, "Open Podman socket SSH tunnel")
		tunnel, err := TunnelRemoteSocketPath(ctx, section, target)
		if err != nil {
			return err
		}
		defer func() {
			stopErr = errors.Join(stopErr, closeRemoteTunnel(tunnel))
		}()

		socket = NewSocket(tunnel.SocketURL())
	}

	section := term.NewSectionPrinter(commandOutput, "Stop services")
	stopErr = RunComposeCommand(ctx, section, socket, scope, "stop")
	return errors.Join(stopErr, commandOutput.Finish())
}
