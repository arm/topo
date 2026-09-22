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
		return command.NewError(cmd, err)
	}
	return nil
}

func ComposeCommand(ctx context.Context, socket Socket, scope project.Scope, args ...string) (*exec.Cmd, error) {
	composeArgs := []string{"compose", "-f", scope.ComposeFile}
	for _, envFile := range scope.EnvFiles {
		composeArgs = append(composeArgs, "--env-file", envFile)
	}
	composeArgs = append(composeArgs, args...)
	return composeCommand(ctx, socket, scope.Env, composeArgs...)
}

// ComposeRawCommand creates a project-independent Compose command without endpoint configuration.
func ComposeRawCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "podman", append([]string{"compose"}, args...)...)
	cmd.Env = append(os.Environ(),
		"PODMAN_COMPOSE_PROVIDER="+composeProvider,
		"PODMAN_COMPOSE_WARNING_LOGS=false",
	)
	return cmd
}

// ComposeSocketRawCommand creates a project-independent Compose command using
// the same provider and endpoint configuration as deployment.
func ComposeSocketRawCommand(ctx context.Context, socket Socket, args ...string) (*exec.Cmd, error) {
	return composeCommand(ctx, socket, nil, append([]string{"compose"}, args...)...)
}

func composeCommand(ctx context.Context, socket Socket, environment []string, args ...string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Env = append(os.Environ(), environment...)
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
		return command.NewError(cmd, err)
	}
	return nil
}
