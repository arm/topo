package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
)

const (
	DefaultRegistryContainerName = "topo-registry"
	DefaultRegistryPort          = "12737"
	tunnelCleanupTimeout         = 5 * time.Second
)

type RegistryConfig struct {
	ContainerName       string
	Port                string
	SkipRemotePortCheck bool
	UseControlSockets   bool
}

type DeployOptions struct {
	RecreateMode docker.RecreateMode
	TargetHost   ssh.Destination
	Registry     *RegistryConfig
}

func SupportsRegistry(noRegistry bool, dest ssh.Destination) bool {
	return !noRegistry && !dest.IsPlainLocalhost()
}

func SupportsSSHControlSockets(goos string) bool {
	return goos != "windows"
}

func Deploy(ctx context.Context, output io.Writer, composeFile string, opts DeployOptions) error {
	sourceHost := command.LocalHost

	if err := term.PrintHeader(output, "Build images"); err != nil {
		return err
	}
	if err := docker.BuildImages(ctx, output, composeFile, sourceHost); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Pull images"); err != nil {
		return err
	}
	if err := docker.PullImages(ctx, output, composeFile, sourceHost); err != nil {
		return err
	}

	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry == nil {
			targetHost := command.NewHostFromDestination(opts.TargetHost)
			if err := transferImagesViaPipe(ctx, output, composeFile, sourceHost, targetHost); err != nil {
				return err
			}
		} else {
			if err := transferImagesViaRegistry(ctx, output, composeFile, sourceHost, opts.TargetHost, *opts.Registry); err != nil {
				return err
			}
		}
	}

	if err := term.PrintHeader(output, "Start services"); err != nil {
		return err
	}
	if err := docker.StartServices(ctx, output, composeFile, command.NewHostFromDestination(opts.TargetHost), opts.RecreateMode); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Deployment Success"); err != nil {
		return err
	}
	return post_deploy.PrintDeploySuccess(output, composeFile, post_deploy.DefaultMessage(composeFile))
}

func transferImagesViaPipe(ctx context.Context, output io.Writer, composeFile string, sourceHost, targetHost command.Host) error {
	if err := term.PrintHeader(output, "Transfer images"); err != nil {
		return err
	}
	return docker.TransferImagesViaPipe(ctx, output, composeFile, sourceHost, targetHost)
}

func transferImagesViaRegistry(ctx context.Context, output io.Writer, composeFile string, sourceHost command.Host, targetHost ssh.Destination, opts RegistryConfig) (transferErr error) {
	if err := term.PrintHeader(output, "Run registry"); err != nil {
		return err
	}
	registryContainerName := opts.ContainerName
	if registryContainerName == "" {
		registryContainerName = DefaultRegistryContainerName
	}
	if err := docker.EnsureRegistryRunning(ctx, output, registryContainerName, opts.Port); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Open registry SSH tunnel"); err != nil {
		return err
	}
	tunnel, err := ssh.OpenSSHTunnel(ctx, output, targetHost, opts.Port, opts.UseControlSockets)
	if err != nil {
		return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with `--registry-port`", err, opts.Port)
	}
	defer func() {
		transferErr = errors.Join(transferErr, closeTunnel(output, tunnel))
	}()

	if !targetHost.IsLocalhost() && !opts.SkipRemotePortCheck {
		if err := term.PrintHeader(output, "Check registry tunnel is not exposed on remote network"); err != nil {
			return err
		}
		if err := docker.CheckTunnelExposure(ctx, output, targetHost, opts.Port); err != nil {
			return err
		}
	}

	if err := term.PrintHeader(output, "Transfer via registry"); err != nil {
		return err
	}
	if err := docker.TransferImagesViaRegistry(ctx, output, composeFile, sourceHost, command.NewHostFromDestination(targetHost), opts.Port); err != nil {
		return err
	}

	return nil
}

func closeTunnel(output io.Writer, tunnel *ssh.SSHTunnel) error {
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
