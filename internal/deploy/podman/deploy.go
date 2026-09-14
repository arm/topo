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

func Deploy(ctx context.Context, output io.Writer, scope project.Scope, options DeployOptions) (deployErr error) {
	commandOutput := term.NewCommandOutput(output)
	if err := EnsureNoRuntimeSet(scope); err != nil {
		return err
	}
	buildOutput := term.NewSectionPrinter(commandOutput, "Build images")
	if err := BuildImages(ctx, buildOutput, LocalSocket, scope); err != nil {
		return err
	}
	pullOutput := term.NewSectionPrinter(commandOutput, "Pull images")
	if err := PullImages(ctx, pullOutput, LocalSocket, scope); err != nil {
		return err
	}

	targetSocket := LocalSocket
	var tunnel *ssh.TCPToUnixSocketTunnel
	if !options.TargetHost.IsPlainLocalhost() {
		tunnelOutput := term.NewSectionPrinter(commandOutput, "Open Podman socket SSH tunnel")

		var err error
		tunnel, err = TunnelRemoteSocketPath(ctx, tunnelOutput, options.TargetHost)
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
			if err := transferImagesViaPipe(ctx, commandOutput, LocalSocket, targetSocket, scope); err != nil {
				return err
			}
		} else if err := transferImagesViaRegistry(ctx, commandOutput, LocalSocket, options.TargetHost, targetSocket, scope, *options.Registry); err != nil {
			return err
		}
	}

	startOutput := term.NewSectionPrinter(commandOutput, "Start services")
	if err := StartServices(ctx, startOutput, targetSocket, scope, options.RecreateMode); err != nil {
		return err
	}

	if tunnel != nil {
		if err := closeRemoteTunnel(tunnel); err != nil {
			return err
		}
		tunnel = nil
	}

	successOutput := term.NewSectionPrinter(commandOutput, "Deployment Success")
	if err := post_deploy.PrintDeploySuccess(successOutput, scope, post_deploy.DefaultMessage(scope.ComposeFile)); err != nil {
		return err
	}
	return commandOutput.Finish()
}

func transferImagesViaPipe(ctx context.Context, output *term.CommandOutput, sourceSocket, targetSocket Socket, scope project.Scope) error {
	section := term.NewSectionPrinter(output, "Transfer images")
	return TransferImagesViaPipe(ctx, section, sourceSocket, targetSocket, scope)
}

func transferImagesViaRegistry(ctx context.Context, output *term.CommandOutput, sourceSocket Socket, targetDestination ssh.Destination, targetSocket Socket, scope project.Scope, options RegistryConfig) (transferErr error) {
	registryOutput := term.NewSectionPrinter(output, "Run registry")
	registryContainerName := options.ContainerName
	if registryContainerName == "" {
		registryContainerName = DefaultRegistryContainerName
	}
	if err := EnsureRegistryRunning(ctx, registryOutput, registryContainerName, options.Port); err != nil {
		return err
	}

	tunnelOutput := term.NewSectionPrinter(output, "Open registry SSH tunnel")
	registryTunnel, err := ssh.OpenTunnel(ctx, tunnelOutput, targetDestination, options.Port)
	if err != nil {
		return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with --registry-port", err, options.Port)
	}
	defer func() {
		transferErr = errors.Join(transferErr, closeRegistryTunnel(output, registryTunnel))
	}()

	if !targetDestination.IsLocalhost() && !options.SkipRemotePortCheck {
		checkOutput := term.NewSectionPrinter(output, "Check registry tunnel is not exposed on remote network")
		if err := deploy.CheckTunnelExposure(ctx, checkOutput, targetDestination, options.Port); err != nil {
			return err
		}
	}

	transferOutput := term.NewSectionPrinter(output, "Transfer via registry")
	return TransferImagesViaRegistry(ctx, transferOutput, sourceSocket, targetSocket, scope, options.Port)
}

func closeRegistryTunnel(output *term.CommandOutput, tunnel *ssh.Tunnel) error {
	ctx, cancel := context.WithTimeout(context.Background(), tunnelCleanupTimeout)
	defer cancel()

	section := term.NewSectionPrinter(output, "Close registry SSH tunnel")
	closeError := tunnel.Close(ctx, section)
	if closeError != nil {
		closeError = fmt.Errorf("failed to close SSH tunnel: %w", closeError)
	}
	return closeError
}

func closeRemoteTunnel(tunnel *ssh.TCPToUnixSocketTunnel) error {
	if err := tunnel.Close(); err != nil {
		return fmt.Errorf("failed to close remote Podman socket tunnel: %w", err)
	}
	return nil
}
