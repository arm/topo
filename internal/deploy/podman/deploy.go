package podman

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
)

type DeployOptions struct {
	TargetHost ssh.Destination
}

func Deploy(ctx context.Context, output io.Writer, composeFile string, options DeployOptions) (deployErr error) {
	if err := term.PrintHeader(output, "Build images"); err != nil {
		return err
	}
	if err := BuildImages(ctx, output, LocalSocket, composeFile); err != nil {
		return err
	}
	if err := term.PrintHeader(output, "Pull images"); err != nil {
		return err
	}
	if err := PullImages(ctx, output, LocalSocket, composeFile); err != nil {
		return err
	}

	targetSocket := LocalSocket
	var tunnel *ssh.TCPToUnixSocketTunnel
	if !options.TargetHost.IsPlainLocalhost() {
		if err := term.PrintHeader(output, "Open Podman socket SSH tunnel"); err != nil {
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
		if err := transferImagesViaPipe(ctx, output, LocalSocket, targetSocket, composeFile); err != nil {
			return err
		}
	}

	if err := term.PrintHeader(output, "Start services"); err != nil {
		return err
	}
	if err := StartServices(ctx, output, targetSocket, composeFile); err != nil {
		return err
	}

	if tunnel != nil {
		if err := closeRemoteTunnel(tunnel); err != nil {
			return err
		}
		tunnel = nil
	}

	if err := term.PrintHeader(output, "Deployment Success"); err != nil {
		return err
	}
	return post_deploy.PrintDeploySuccess(output, composeFile, options.TargetHost, post_deploy.DefaultMessage(composeFile))
}

func transferImagesViaPipe(ctx context.Context, output io.Writer, source, destination Socket, composeFile string) error {
	if err := term.PrintHeader(output, "Transfer images"); err != nil {
		return err
	}
	return TransferImagesViaPipe(ctx, output, source, destination, composeFile)
}

func closeRemoteTunnel(tunnel *ssh.TCPToUnixSocketTunnel) error {
	if err := tunnel.Close(); err != nil {
		return fmt.Errorf("failed to close remote Podman socket tunnel: %w", err)
	}
	return nil
}
