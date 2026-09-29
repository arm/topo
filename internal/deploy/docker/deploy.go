package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

const (
	DefaultRegistryContainerName = "topo-registry"
	DefaultRegistryPort          = "12737"
	tunnelCleanupTimeout         = 5 * time.Second
)

func Deploy(ctx context.Context, output io.Writer, scope project.Scope, opts deploy.Options) error {
	sourceHost := LocalHost
	progress := term.NewProgress(output)

	if err := progress.Header("Build images"); err != nil {
		return err
	}
	if err := BuildImages(ctx, output, sourceHost, scope); err != nil {
		return err
	}

	if err := progress.Header("Pull images"); err != nil {
		return err
	}
	if err := PullImages(ctx, output, sourceHost, scope); err != nil {
		return err
	}

	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry == nil {
			targetHost := NewHostFromDestination(opts.TargetHost)
			if err := transferImagesViaPipe(ctx, progress, sourceHost, targetHost, scope); err != nil {
				return err
			}
		} else {
			if err := transferImagesViaRegistry(ctx, progress, sourceHost, opts.TargetHost, scope, *opts.Registry); err != nil {
				return err
			}
		}
	}

	if err := progress.Header("Start services"); err != nil {
		return err
	}
	if err := StartServices(ctx, output, NewHostFromDestination(opts.TargetHost), scope, opts.RecreateMode); err != nil {
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

func transferImagesViaPipe(ctx context.Context, progress *term.Progress, sourceHost, targetHost Host, scope project.Scope) error {
	if err := progress.Header("Transfer images"); err != nil {
		return err
	}
	return TransferImagesViaPipe(ctx, progress.Output(), sourceHost, targetHost, scope)
}

func transferImagesViaRegistry(ctx context.Context, progress *term.Progress, sourceHost Host, targetHost ssh.Destination, scope project.Scope, opts deploy.RegistryConfig) (transferErr error) {
	output := progress.Output()
	if err := progress.Header("Run registry"); err != nil {
		return err
	}
	registryContainerName := opts.ContainerName
	if registryContainerName == "" {
		registryContainerName = DefaultRegistryContainerName
	}
	if err := EnsureRegistryRunning(ctx, output, registryContainerName, opts.Port); err != nil {
		return err
	}

	if err := progress.Header("Open registry SSH tunnel"); err != nil {
		return err
	}
	tunnel, err := ssh.OpenTunnel(ctx, output, targetHost, opts.Port)
	if err != nil {
		return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with `--registry-port`", err, opts.Port)
	}
	defer func() {
		transferErr = errors.Join(transferErr, closeTunnel(progress, tunnel))
	}()

	if !targetHost.IsLocalhost() && !opts.SkipRemotePortCheck {
		if err := progress.Header("Check registry tunnel is not exposed on remote network"); err != nil {
			return err
		}
		if err := deploy.CheckTunnelExposure(ctx, output, targetHost, opts.Port); err != nil {
			return err
		}
	}

	if err := progress.Header("Transfer via registry"); err != nil {
		return err
	}
	if err := TransferImagesViaRegistry(ctx, output, sourceHost, NewHostFromDestination(targetHost), scope, opts.Port); err != nil {
		return err
	}

	return nil
}

func closeTunnel(progress *term.Progress, tunnel *ssh.Tunnel) error {
	ctx, cancel := context.WithTimeout(context.Background(), tunnelCleanupTimeout)
	defer cancel()

	output := progress.Output()
	var headerError error
	if output != nil {
		headerError = progress.Header("Close registry SSH tunnel")
	}
	closeError := tunnel.Close(ctx, output)
	if closeError != nil {
		closeError = fmt.Errorf("failed to close SSH tunnel: %w", closeError)
	}
	return errors.Join(headerError, closeError)
}
