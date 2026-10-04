package podman

import (
	"context"
	"io"
	"os/exec"
)

type EngineExecutor struct {
	socket Socket
}

func (ex EngineExecutor) Command(ctx context.Context, args ...string) *exec.Cmd {
	return Command(ctx, ex.socket, args...)
}

func (ex EngineExecutor) RunCommand(ctx context.Context, output io.Writer, args ...string) error {
	return RunCommand(ctx, output, ex.socket, args...)
}
