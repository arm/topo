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
}

type DeployOptions struct {
	RecreateMode RecreateMode
	TargetHost   ssh.Destination
	Registry     *RegistryConfig
}

func SupportsRegistry(noRegistry bool, dest ssh.Destination) bool {
	return !noRegistry && !dest.IsPlainLocalhost()
}

func Deploy(ctx context.Context, output io.Writer, composeFile string, opts DeployOptions) error {
	sourceHost := LocalHost

	if err := term.PrintHeader(output, "Build images"); err != nil {
		return err
	}
	if err := BuildImages(ctx, output, sourceHost, composeFile); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Pull images"); err != nil {
		return err
	}
	if err := PullImages(ctx, output, sourceHost, composeFile); err != nil {
		return err
	}

	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry == nil {
			targetHost := NewHostFromDestination(opts.TargetHost)
			if err := transferImagesViaPipe(ctx, output, sourceHost, targetHost, composeFile); err != nil {
				return err
			}
		} else {
			if err := transferImagesViaRegistry(ctx, output, sourceHost, opts.TargetHost, composeFile, *opts.Registry); err != nil {
				return err
			}
		}
	}

	if err := term.PrintHeader(output, "Start services"); err != nil {
		return err
	}
	if err := StartServices(ctx, output, NewHostFromDestination(opts.TargetHost), composeFile, opts.RecreateMode); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Deployment Success"); err != nil {
		return err
	}
	return post_deploy.PrintDeploySuccess(output, composeFile, post_deploy.DefaultMessage(composeFile))
}

func transferImagesViaPipe(ctx context.Context, output io.Writer, sourceHost, targetHost Host, composeFile string) error {
	if err := term.PrintHeader(output, "Transfer images"); err != nil {
		return err
	}
	return TransferImagesViaPipe(ctx, output, sourceHost, targetHost, composeFile)
}

func transferImagesViaRegistry(ctx context.Context, output io.Writer, sourceHost Host, targetHost ssh.Destination, composeFile string, opts RegistryConfig) (transferErr error) {
	if err := term.PrintHeader(output, "Run registry"); err != nil {
		return err
	}
	registryContainerName := opts.ContainerName
	if registryContainerName == "" {
		registryContainerName = DefaultRegistryContainerName
	}
	if err := EnsureRegistryRunning(ctx, output, registryContainerName, opts.Port); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Open registry SSH tunnel"); err != nil {
		return err
	}
	tunnel, err := ssh.OpenTunnel(ctx, output, targetHost, opts.Port)
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
		if err := deploy.CheckTunnelExposure(ctx, output, targetHost, opts.Port); err != nil {
			return err
		}
	}

	if err := term.PrintHeader(output, "Transfer via registry"); err != nil {
		return err
	}
	if err := TransferImagesViaRegistry(ctx, output, sourceHost, NewHostFromDestination(targetHost), composeFile, opts.Port); err != nil {
		return err
	}

	return nil
}

func closeTunnel(output io.Writer, tunnel *ssh.Tunnel) error {
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
