package deploy

import (
	"context"
	"fmt"
	"io"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"golang.org/x/sync/errgroup"
)

func PrepareImages(ctx context.Context, progress *term.Progress, scope project.Scope, runCompose RunComposeCommandFn) error {
	if err := BuildImages(ctx, progress, scope, runCompose); err != nil {
		return err
	}
	return PullImages(ctx, progress, scope, runCompose)
}

type RunSaveCommandFn func(ctx context.Context, output io.Writer, image string, imagePayload io.Writer) error

type RunLoadCommandFn func(ctx context.Context, output io.Writer, imagePayload io.Reader) error

func TransferImagesViaPipe(
	ctx context.Context,
	progress *term.Progress,
	runSave RunSaveCommandFn,
	runLoad RunLoadCommandFn,
	scope project.Scope,
) error {
	if err := progress.Header("Transfer images"); err != nil {
		return err
	}

	images, err := project.ImageNames(scope)
	if err != nil {
		return err
	}

	var group errgroup.Group
	for _, image := range images {
		group.Go(func() error {
			return transferImageViaPipe(ctx, progress.Output(), runSave, runLoad, image)
		})
	}
	return group.Wait()
}

func transferImageViaPipe(
	ctx context.Context,
	output io.Writer,
	runSave RunSaveCommandFn,
	runLoad RunLoadCommandFn,
	image string,
) error {
	pipeReader, pipeWriter := io.Pipe()

	var group errgroup.Group
	group.Go(func() error {
		err := runSave(ctx, output, image, pipeWriter)
		_ = pipeWriter.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to save image %s: %w", image, err)
		}
		return nil
	})
	group.Go(func() error {
		err := runLoad(ctx, output, pipeReader)
		_ = pipeReader.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to load image %s: %w", image, err)
		}
		return nil
	})
	return group.Wait()
}
