package docker

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func Stop(ctx context.Context, output io.Writer, scope project.Scope, destination ssh.Destination) error {
	progress := term.NewProgress(output)
	remoteEngine := EngineExecutor{NewHostFromDestination(destination)}
	return deploy.StopServices(ctx, progress, scope, remoteEngine)
}
