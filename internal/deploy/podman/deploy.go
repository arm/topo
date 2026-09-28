package podman

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
	tunnelCleanupTimeout         = 5 * time.Second
)

func Deploy(ctx context.Context, output io.Writer, scope project.Scope, options deploy.Options) (deployErr error) {
	if err := EnsureNoRuntimeSet(scope); err != nil {
		return err
	}
	progress := term.NewProgress(output)
	if err := progress.Header("Build images"); err != nil {
		return err
	}
	if err := BuildImages(ctx, output, LocalSocket, scope); err != nil {
		return err
	}
	if err := progress.Header("Pull images"); err != nil {
		return err
	}
	if err := PullImages(ctx, output, LocalSocket, scope); err != nil {
		return err
	}

	targetSocket := LocalSocket
	var tunnel *ssh.TCPToUnixSocketTunnel
	if !options.TargetHost.IsPlainLocalhost() {
		if err := progress.Header("Open Podman socket SSH tunnel"); err != nil {
			return err
		}

		var err error
		tunnel, err = TunnelRemoteSocketPath(ctx, output, options.TargetHost)
		if err != nil {
			return err
		}
		defer func() {
			if tunnel != nil {
				deployErr = errors.Join(deployErr, closeRemoteTunnel(tunnel))
			}
		}()

		targetSocket = NewSocket(tunnel.SocketURL())
		if options.Registry == nil {
			if err := transferImagesViaPipe(ctx, progress, LocalSocket, targetSocket, scope); err != nil {
				return err
			}
		} else if err := transferImagesViaRegistry(ctx, progress, LocalSocket, options.TargetHost, targetSocket, scope, *options.Registry); err != nil {
			return err
		}
	}

	if err := progress.Header("Start services"); err != nil {
		return err
	}
	if err := StartServices(ctx, output, targetSocket, scope, options.RecreateMode); err != nil {
		return err
	}

	if tunnel != nil {
		if err := closeRemoteTunnel(tunnel); err != nil {
			return err
		}
		tunnel = nil
	}

	if err := progress.Header("Deployment Success"); err != nil {
		return err
	}
	return post_deploy.PrintDeploySuccess(
		output,
		scope,
		options.DefaultSuccessMessage,
	)
}

func transferImagesViaPipe(ctx context.Context, progress *term.Progress, sourceSocket, targetSocket Socket, scope project.Scope) error {
	if err := progress.Header("Transfer images"); err != nil {
		return err
	}
	return TransferImagesViaPipe(ctx, progress.Output(), sourceSocket, targetSocket, scope)
}

func transferImagesViaRegistry(ctx context.Context, progress *term.Progress, sourceSocket Socket, targetDestination ssh.Destination, targetSocket Socket, scope project.Scope, options deploy.RegistryConfig) (transferErr error) {
	output := progress.Output()
	if err := progress.Header("Run registry"); err != nil {
		return err
	}
	registryContainerName := options.ContainerName
	if registryContainerName == "" {
		registryContainerName = DefaultRegistryContainerName
	}
	if err := EnsureRegistryRunning(ctx, output, registryContainerName, options.Port); err != nil {
		return err
	}

	if err := progress.Header("Open registry SSH tunnel"); err != nil {
		return err
	}
	registryTunnel, err := ssh.OpenTunnel(ctx, output, targetDestination, options.Port)
	if err != nil {
		return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with --registry-port", err, options.Port)
	}
	defer func() {
		transferErr = errors.Join(transferErr, closeRegistryTunnel(progress, registryTunnel))
	}()

	if !targetDestination.IsLocalhost() && !options.SkipRemotePortCheck {
		if err := progress.Header("Check registry tunnel is not exposed on remote network"); err != nil {
			return err
		}
		if err := deploy.CheckTunnelExposure(ctx, output, targetDestination, options.Port); err != nil {
			return err
		}
	}

	if err := progress.Header("Transfer via registry"); err != nil {
		return err
	}
	return TransferImagesViaRegistry(ctx, output, sourceSocket, targetSocket, scope, options.Port)
}

func closeRegistryTunnel(progress *term.Progress, tunnel *ssh.Tunnel) error {
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

func closeRemoteTunnel(tunnel *ssh.TCPToUnixSocketTunnel) error {
	if err := tunnel.Close(); err != nil {
		return fmt.Errorf("failed to close remote Podman socket tunnel: %w", err)
	}
	return nil
}
