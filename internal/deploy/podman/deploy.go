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
	"github.com/arm/topo/internal/ssh"
)

const (
	DefaultRegistryContainerName = "topo-registry"
	tunnelCleanupTimeout         = 5 * time.Second
)

type RegistryConfig struct {
	ContainerName       string
	Port                string
	SkipRemotePortCheck bool
}

type DeployOptions struct {
	RecreateMode RecreateMode
	TargetHost   ssh.Destination
	Registry     *RegistryConfig
}

func Deploy(ctx context.Context, output io.Writer, composeFile string, options DeployOptions) (deployErr error) {
	if err := EnsureNoRuntimeSet(composeFile); err != nil {
		return err
	}
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
		if options.Registry == nil {
			if err := transferImagesViaPipe(ctx, output, LocalSocket, targetSocket, composeFile); err != nil {
				return err
			}
		} else if err := transferImagesViaRegistry(ctx, output, LocalSocket, options.TargetHost, targetSocket, composeFile, *options.Registry); err != nil {
			return err
		}
	}

	if err := term.PrintHeader(output, "Start services"); err != nil {
		return err
	}
	if err := StartServices(ctx, output, targetSocket, composeFile, options.RecreateMode); err != nil {
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
	return post_deploy.PrintDeploySuccess(output, composeFile, post_deploy.DefaultMessage(composeFile))
}

func transferImagesViaPipe(ctx context.Context, output io.Writer, sourceSocket, targetSocket Socket, composeFile string) error {
	if err := term.PrintHeader(output, "Transfer images"); err != nil {
		return err
	}
	return TransferImagesViaPipe(ctx, output, sourceSocket, targetSocket, composeFile)
}

func transferImagesViaRegistry(ctx context.Context, output io.Writer, sourceSocket Socket, targetDestination ssh.Destination, targetSocket Socket, composeFile string, options RegistryConfig) (transferErr error) {
	if err := term.PrintHeader(output, "Run registry"); err != nil {
		return err
	}
	registryContainerName := options.ContainerName
	if registryContainerName == "" {
		registryContainerName = DefaultRegistryContainerName
	}
	if err := EnsureRegistryRunning(ctx, output, registryContainerName, options.Port); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Open registry SSH tunnel"); err != nil {
		return err
	}
	registryTunnel, err := ssh.OpenTunnel(ctx, output, targetDestination, options.Port)
	if err != nil {
		return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with --registry-port", err, options.Port)
	}
	defer func() {
		transferErr = errors.Join(transferErr, closeRegistryTunnel(output, registryTunnel))
	}()

	if !targetDestination.IsLocalhost() && !options.SkipRemotePortCheck {
		if err := term.PrintHeader(output, "Check registry tunnel is not exposed on remote network"); err != nil {
			return err
		}
		if err := deploy.CheckTunnelExposure(ctx, output, targetDestination, options.Port); err != nil {
			return err
		}
	}

	if err := term.PrintHeader(output, "Transfer via registry"); err != nil {
		return err
	}
	return TransferImagesViaRegistry(ctx, output, sourceSocket, targetSocket, composeFile, options.Port)
}

func closeRegistryTunnel(output io.Writer, tunnel *ssh.Tunnel) error {
	ctx, cancel := context.WithTimeout(context.Background(), tunnelCleanupTimeout)
	defer cancel()

	var headerError error
	if output != nil {
		headerError = term.PrintHeader(output, "Close registry SSH tunnel")
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
