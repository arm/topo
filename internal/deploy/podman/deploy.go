package podman

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func Deploy(ctx context.Context, output io.Writer, scope project.Scope, options deploy.Options) (deployErr error) {
	if err := EnsureNoRuntimeSet(scope); err != nil {
		return err
	}
	progress := term.NewProgress(output)
	localEngine := EngineExecutor{LocalSocket}

	if err := deploy.PrepareImages(ctx, progress, scope, localEngine); err != nil {
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

	remoteEngine := EngineExecutor{targetSocket}
	if err := deploy.StartServices(ctx, progress, scope, options.RecreateMode, remoteEngine); err != nil {
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
	return deploy.TransferImagesViaPipe(
		ctx,
		progress,
		func(ctx context.Context, output io.Writer, image string, imagePayload io.Writer) error {
			saveCommand := Command(ctx, sourceSocket, "save", image)
			saveCommand.Stdout = imagePayload
			saveCommand.Stderr = output
			return saveCommand.Run()
		},
		func(ctx context.Context, output io.Writer, imagePayload io.Reader) error {
			loadCommand := Command(ctx, targetSocket, "load")
			loadCommand.Stdin = imagePayload
			loadCommand.Stderr = output
			loadCommand.Stdout = output
			return loadCommand.Run()
		},
		scope,
	)
}

func transferImagesViaRegistry(ctx context.Context, progress *term.Progress, sourceSocket Socket, targetDestination ssh.Destination, targetSocket Socket, scope project.Scope, options deploy.RegistryConfig) (transferErr error) {
	options = options.WithDefaults()

	if err := deploy.PrepareRegistry(ctx, progress, options,
		EngineExecutor{socket: sourceSocket},
		// pasta reports either "Address in use" or "Address already in use",
		// while Podman machine's macOS port-forwarding proxy reports "proxy already running".
		[]string{"address in use", "address already in use", "proxy already running"},
	); err != nil {
		return err
	}

	if closeRegistryTunnel, err := deploy.OpenRegistrySSHTunnel(ctx, progress, targetDestination, options); err != nil {
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
		options.Port,
		EngineExecutor{socket: sourceSocket},
		EngineExecutor{socket: targetSocket},
	)
}

func closeRemoteTunnel(tunnel *ssh.TCPToUnixSocketTunnel) error {
	if err := tunnel.Close(); err != nil {
		return fmt.Errorf("failed to close remote Podman socket tunnel: %w", err)
	}
	return nil
}
