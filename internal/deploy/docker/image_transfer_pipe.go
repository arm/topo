package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/arm/topo/internal/compose"
	"golang.org/x/sync/errgroup"
)

func TransferImagesViaPipe(ctx context.Context, output io.Writer, source, destination Host, composeFile string) error {
	images, err := compose.DockerImageNames(composeFile)
	if err != nil {
		return err
	}

	var group errgroup.Group
	for _, image := range images {
		group.Go(func() error {
			return transferImageViaPipe(ctx, output, source, destination, image)
		})
	}
	return group.Wait()
}

func transferImageViaPipe(ctx context.Context, output io.Writer, source, destination Host, image string) error {
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
		defer pipeWriter.Close() //nolint:errcheck
		if err := saveCommand.Run(); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		defer pipeReader.Close() //nolint:errcheck
		if err := loadCommand.Run(); err != nil {
			return fmt.Errorf("failed to load image: %w", err)
		}
		return nil
	})
	return group.Wait()
}
