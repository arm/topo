package podman

import (
	"context"
	"errors"
	"io"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func Stop(ctx context.Context, output io.Writer, scope project.Scope, target ssh.Destination) (stopErr error) {
	socket := LocalSocket
	progress := term.NewProgress(output)
	if !target.IsPlainLocalhost() {
		if err := progress.Header("Open Podman socket SSH tunnel"); err != nil {
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

	remoteEngine := EngineExecutor{socket}
	return deploy.StopServices(ctx, progress, scope, remoteEngine)
}
