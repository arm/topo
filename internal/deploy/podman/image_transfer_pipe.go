package podman

import (
	"context"
	"fmt"
	"io"

	"github.com/arm/topo/internal/compose"
	"golang.org/x/sync/errgroup"
)

func TransferImagesViaPipe(ctx context.Context, output io.Writer, source, destination Socket, composeFile string) error {
	images, err := compose.ImageNames(composeFile)
	if err != nil {
		return err
	}
	for _, image := range images {
		if err := transferImageViaPipe(ctx, output, source, destination, image); err != nil {
			return err
		}
	}
	return nil
}

func transferImageViaPipe(ctx context.Context, output io.Writer, source, destination Socket, image string) error {
	pipeReader, pipeWriter := io.Pipe()
	saveCommand := Command(ctx, source, "save", image)
	loadCommand := Command(ctx, destination, "load")
	saveCommand.Stdout = pipeWriter
	saveCommand.Stderr = output
	loadCommand.Stdin = pipeReader
	loadCommand.Stdout = output
	loadCommand.Stderr = output

	var group errgroup.Group
	group.Go(func() error {
		err := saveCommand.Run()
		_ = pipeWriter.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to save image %s: %w", image, err)
		}
		return nil
	})
	group.Go(func() error {
		err := loadCommand.Run()
		_ = pipeReader.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to load image %s: %w", image, err)
		}
		return nil
	})
	return group.Wait()
}
