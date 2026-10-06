package docker

import (
	"context"
	"io"
	"os/exec"
)

type EngineExecutor struct {
	host Host
}

func (ex EngineExecutor) Command(ctx context.Context, args ...string) *exec.Cmd {
	return Command(ctx, ex.host, args...)
}

func (ex EngineExecutor) RunCommand(ctx context.Context, output io.Writer, args ...string) error {
	return RunCommand(ctx, output, ex.host, args...)
}
