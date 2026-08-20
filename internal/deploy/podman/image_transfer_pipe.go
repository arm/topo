package podman

import (
	"context"
	"fmt"
	"io"

	"github.com/arm/topo/internal/compose"
	"golang.org/x/sync/errgroup"
)

func TransferImagesViaPipe(ctx context.Context, output io.Writer, sourceSocket, targetSocket Socket, composeFile string) error {
	images, err := compose.ImageNames(composeFile)
	if err != nil {
		return err
	}

	var group errgroup.Group
	for _, image := range images {
		group.Go(func() error {
			return transferImageViaPipe(ctx, output, sourceSocket, targetSocket, image)
		})
	}
	return group.Wait()
}

func transferImageViaPipe(ctx context.Context, output io.Writer, sourceSocket, targetSocket Socket, image string) error {
	pipeReader, pipeWriter := io.Pipe()
	saveCommand := Command(ctx, sourceSocket, "save", image)
	loadCommand := Command(ctx, targetSocket, "load")
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
