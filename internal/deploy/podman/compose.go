package podman

import (
	"context"
	"io"

	"github.com/arm/topo/internal/compose"
)

func BuildImages(ctx context.Context, output io.Writer, composeFile string) error {
	return RunComposeCommand(ctx, output, composeFile, "build")
}

func PullImages(ctx context.Context, output io.Writer, composeFile string) error {
	services, err := compose.PullableServices(composeFile)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return nil
	}

	args := append([]string{"pull"}, services...)
	return RunComposeCommand(ctx, output, composeFile, args...)
}

func StartServices(ctx context.Context, output io.Writer, composeFile string) error {
	return RunComposeCommand(ctx, output, composeFile, "up", "-d", "--no-build", "--pull", "never")
}
