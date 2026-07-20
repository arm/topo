package docker

import (
	"context"
	"io"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/deploy/command"
)

type RecreateMode int

const (
	RecreateModeDefault RecreateMode = iota
	RecreateModeForce
	RecreateModeNone
)

func BuildImages(ctx context.Context, output io.Writer, composeFile string, host command.Host) error {
	return runCompose(ctx, host, composeFile, output, "build")
}

func PullImages(ctx context.Context, output io.Writer, composeFile string, host command.Host) error {
	services, err := compose.PullableServices(composeFile)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return nil
	}

	args := append([]string{"pull"}, services...)
	return runCompose(ctx, host, composeFile, output, args...)
}

func StartServices(ctx context.Context, output io.Writer, composeFile string, host command.Host, mode RecreateMode) error {
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	switch mode {
	case RecreateModeForce:
		args = append(args, "--force-recreate")
	case RecreateModeNone:
		args = append(args, "--no-recreate")
	}
	return runCompose(ctx, host, composeFile, output, args...)
}

func StopServices(ctx context.Context, output io.Writer, composeFile string, host command.Host) error {
	return runCompose(ctx, host, composeFile, output, "stop")
}

func runCompose(ctx context.Context, host command.Host, composeFile string, output io.Writer, args ...string) error {
	cmd := command.DockerComposeContext(ctx, host, composeFile, args...)
	cmd.Stdout = output
	cmd.Stderr = output
	return cmd.Run()
}
