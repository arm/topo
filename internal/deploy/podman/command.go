package podman

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/arm/topo/internal/command"
)

const composeProvider = "docker-compose"

func Command(ctx context.Context, socket Socket, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Env = socket.ConfigurePodmanEnv(os.Environ())
	return cmd
}

func ComposeCommand(ctx context.Context, socket Socket, composeFile string, args ...string) (*exec.Cmd, error) {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmd := exec.CommandContext(ctx, "podman", composeArgs...)
	cmd.Env = append(os.Environ(),
		"PODMAN_COMPOSE_PROVIDER="+composeProvider,
		"PODMAN_COMPOSE_WARNING_LOGS=false",
	)
	var err error
	cmd.Env, err = socket.ConfigureComposeEnv(cmd.Env)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func RunComposeCommand(ctx context.Context, output io.Writer, socket Socket, composeFile string, args ...string) error {
	cmd, err := ComposeCommand(ctx, socket, composeFile, args...)
	if err != nil {
		return err
	}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return command.FormatError(cmd.Args, err)
	}
	return nil
}
