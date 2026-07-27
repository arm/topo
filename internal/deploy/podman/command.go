package podman

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/arm/topo/internal/command"
)

const composeProvider = "docker-compose"

func Command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "podman", args...)
}

func ComposeCommand(ctx context.Context, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmd := exec.CommandContext(ctx, "podman", composeArgs...)
	cmd.Env = append(os.Environ(),
		"PODMAN_COMPOSE_PROVIDER="+composeProvider,
		"PODMAN_COMPOSE_WARNING_LOGS=false",
	)
	return cmd
}

func RunComposeCommand(ctx context.Context, output io.Writer, composeFile string, args ...string) error {
	cmd := ComposeCommand(ctx, composeFile, args...)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return command.FormatError(cmd.Args, err)
	}
	return nil
}
