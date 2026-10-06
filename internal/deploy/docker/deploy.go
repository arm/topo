package docker

import (
	"context"
	"errors"
	"io"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func Deploy(ctx context.Context, output io.Writer, scope project.Scope, opts deploy.Options) error {
	sourceHost := LocalHost
	localEngine := EngineExecutor{LocalHost}
	progress := term.NewProgress(output)

	if err := deploy.PrepareImages(ctx, progress, scope, localEngine); err != nil {
		return err
	}

	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry == nil {
			targetHost := NewHostFromDestination(opts.TargetHost)
			if err := deploy.TransferImagesViaPipe(ctx, progress, scope, localEngine, EngineExecutor{targetHost}); err != nil {
				return err
			}
		} else {
			if err := transferImagesViaRegistry(ctx, progress, sourceHost, opts.TargetHost, scope, *opts.Registry); err != nil {
				return err
			}
		}
	}

	remoteEngine := EngineExecutor{NewHostFromDestination(opts.TargetHost)}
	if err := deploy.StartServices(ctx, progress, scope, opts.RecreateMode, remoteEngine); err != nil {
		return err
	}

	if err := progress.Header("Deployment Success"); err != nil {
		return err
	}
	return post_deploy.PrintDeploySuccess(
		output,
		scope,
		opts.DefaultSuccessMessage,
	)
}

func transferImagesViaRegistry(ctx context.Context, progress *term.Progress, sourceHost Host, targetHost ssh.Destination, scope project.Scope, opts deploy.RegistryConfig) (transferErr error) {
	opts = opts.WithDefaults()

	if err := deploy.PrepareRegistry(ctx, progress, opts,
		EngineExecutor{host: LocalHost},
		[]string{"already in use", "already allocated"},
	); err != nil {
		return err
	}

	if closeRegistryTunnel, err := deploy.OpenRegistrySSHTunnel(ctx, progress, targetHost, opts); err != nil {
		return err
	} else {
		defer func() {
			transferErr = errors.Join(transferErr, closeRegistryTunnel(progress))
		}()
	}

	return deploy.TransferImagesViaRegistry(
		ctx,
		progress,
		scope,
		opts.Port,
		EngineExecutor{host: sourceHost},
		EngineExecutor{host: NewHostFromDestination(targetHost)},
	)
}
