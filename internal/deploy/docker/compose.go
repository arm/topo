package docker

import (
	"context"
	"io"

	"github.com/arm/topo/internal/compose"
)

type RecreateMode int

const (
	RecreateModeDefault RecreateMode = iota
	RecreateModeForce
	RecreateModeNone
)

func BuildImages(ctx context.Context, output io.Writer, host Host, composeFile string) error {
	return RunComposeCommand(ctx, output, host, composeFile, "build")
}

func PullImages(ctx context.Context, output io.Writer, host Host, composeFile string) error {
	services, err := compose.PullableServices(composeFile)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return nil
	}

	args := append([]string{"pull"}, services...)
	return RunComposeCommand(ctx, output, host, composeFile, args...)
}

func StartServices(ctx context.Context, output io.Writer, host Host, composeFile string, mode RecreateMode) error {
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	switch mode {
	case RecreateModeForce:
		args = append(args, "--force-recreate")
	case RecreateModeNone:
		args = append(args, "--no-recreate")
	}
	return RunComposeCommand(ctx, output, host, composeFile, args...)
}

func StopServices(ctx context.Context, output io.Writer, host Host, composeFile string) error {
	return RunComposeCommand(ctx, output, host, composeFile, "stop")
}
