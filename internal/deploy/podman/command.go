package podman

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/project"
)

const composeProvider = "docker-compose"

func Command(ctx context.Context, socket Socket, args ...string) *exec.Cmd {
	// #nosec G702 -- Podman arguments are passed directly, not interpreted by a shell.
	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Env = socket.ConfigurePodmanEnv(os.Environ())
	return cmd
}

func RunCommand(ctx context.Context, output io.Writer, socket Socket, args ...string) error {
	cmd := Command(ctx, socket, args...)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return command.FormatError(cmd.Args, err)
	}
	return nil
}

func ComposeCommand(ctx context.Context, socket Socket, scope project.Scope, args ...string) (*exec.Cmd, error) {
	composeArgs := []string{"compose", "-f", scope.ComposeFile}
	for _, envFile := range scope.EnvFiles {
		composeArgs = append(composeArgs, "--env-file", envFile)
	}
	composeArgs = append(composeArgs, args...)
	cmd := exec.CommandContext(ctx, "podman", composeArgs...)
	cmd.Env = append(os.Environ(), scope.Env...)
	cmd.Env = append(cmd.Env,
		"PODMAN_COMPOSE_PROVIDER="+composeProvider,
		"PODMAN_COMPOSE_WARNING_LOGS=false",
	)
	var err error
	cmd.Env, err = socket.ConfigureComposeEnv(ctx, cmd.Env)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func RunComposeCommand(ctx context.Context, output io.Writer, socket Socket, scope project.Scope, args ...string) error {
	cmd, err := ComposeCommand(ctx, socket, scope, args...)
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
