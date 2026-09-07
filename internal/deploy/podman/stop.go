package podman

import (
	"context"
	"errors"
	"io"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
)

func Stop(ctx context.Context, output io.Writer, composeFile string, target ssh.Destination) (stopErr error) {
	socket := LocalSocket
	if !target.IsPlainLocalhost() {
		if err := term.PrintHeader(output, "Open Podman socket SSH tunnel"); err != nil {
			return err
		}

		tunnel, err := TunnelRemoteSocketPath(ctx, output, target)
		if err != nil {
			return err
		}
		defer func() {
			stopErr = errors.Join(stopErr, closeRemoteTunnel(tunnel))
		}()

		socket = NewSocket(tunnel.SocketURL())
	}

	if err := term.PrintHeader(output, "Stop services"); err != nil {
		return err
	}
	return RunComposeCommand(ctx, output, socket, composeFile, "stop")
}
