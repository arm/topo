package docker

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func Command(ctx context.Context, host Host, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(host), args...)
	return exec.CommandContext(ctx, "docker", cmdArgs...)
}

func RunCommand(ctx context.Context, host Host, output io.Writer, args ...string) error {
	return run(Command(ctx, host, args...), output)
}

func String(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}

func ComposeCommand(ctx context.Context, host Host, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdArgs := append(hostToArgs(host), composeArgs...)
	return exec.CommandContext(ctx, "docker", cmdArgs...)
}

func RunComposeCommand(ctx context.Context, host Host, composeFile string, output io.Writer, args ...string) error {
	return run(ComposeCommand(ctx, host, composeFile, args...), output)
}

func run(cmd *exec.Cmd, output io.Writer) error {
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s: %w", String(cmd), err)
	}
	return nil
}

func hostToArgs(h Host) []string {
	if h.value == "" {
		return nil
	}
	return []string{"-H", h.value}
}
