package docker

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/project"
)

func Command(ctx context.Context, host Host, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(host), args...)
	return exec.CommandContext(ctx, "docker", cmdArgs...)
}

func RunCommand(ctx context.Context, output io.Writer, host Host, args ...string) error {
	return run(Command(ctx, host, args...), output)
}

func ComposeCommand(ctx context.Context, host Host, scope project.Scope, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", scope.ComposeFile}, args...)
	cmdArgs := append(hostToArgs(host), composeArgs...)
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Env = append(os.Environ(), scope.Env...)
	return cmd
}

func RunComposeCommand(ctx context.Context, output io.Writer, host Host, scope project.Scope, args ...string) error {
	return run(ComposeCommand(ctx, host, scope, args...), output)
}

func run(cmd *exec.Cmd, output io.Writer) error {
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return command.FormatError(cmd.Args, err)
	}
	return nil
}

func hostToArgs(h Host) []string {
	if h.value == "" {
		return nil
	}
	return []string{"-H", h.value}
}
