package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

type LogsOptions struct {
	Services []string
	Follow   bool
}

func Logs(
	ctx context.Context,
	output io.Writer,
	scope project.Scope,
	target ssh.Destination,
	engine EngineKind,
	options LogsOptions,
) (logsErr error) {
	if engine != EngineDocker && engine != EnginePodman {
		return fmt.Errorf("invalid engine %q: must be docker or podman", engine)
	}

	if engine == EngineDocker {
		runner := docker.Engine{Host: docker.NewHostFromDestination(target)}
		return LogServices(ctx, output, scope, runner, options)
	}

	socket := podman.LocalSocket
	if !target.IsPlainLocalhost() {
		tunnel, err := podman.TunnelRemoteSocketPath(ctx, io.Discard, target)
		if err != nil {
			return err
		}
		defer func() {
			logsErr = errors.Join(logsErr, tunnel.Close())
		}()
		socket = podman.NewSocket(tunnel.SocketURL())
	}
	return LogServices(ctx, output, scope, podman.Engine{Socket: socket}, options)
}
