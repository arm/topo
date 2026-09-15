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
	socket := LocalSocket
	printHeader := term.PrintFirstHeader
	if !target.IsPlainLocalhost() {
		if err := printHeader(output, "Open Podman socket SSH tunnel"); err != nil {
			return err
		}
		printHeader = term.PrintNthHeader

		tunnel, err := TunnelRemoteSocketPath(ctx, output, target)
		if err != nil {
			return err
		}
		defer func() {
			stopErr = errors.Join(stopErr, closeRemoteTunnel(tunnel))
		}()

		socket = NewSocket(tunnel.SocketURL())
	}

	if err := printHeader(output, "Stop services"); err != nil {
		return err
	}
	return RunComposeCommand(ctx, output, socket, scope, "stop")
}
