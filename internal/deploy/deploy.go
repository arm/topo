package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/operation"
	"github.com/arm/topo/internal/deploy/post_deploy"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
)

const (
	DefaultRegistryContainerName = "topo-registry"
	DefaultRegistryPort          = "12737"
	tunnelCleanupTimeout         = 5 * time.Second
)

type RegistryConfig struct {
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

func NewDeploymentStop(composeFile string, dest ssh.Destination) goperation.Sequence {
	return goperation.Sequence{operation.NewDockerComposeStop(composeFile, command.NewHostFromDestination(dest))}
}

func Deploy(ctx context.Context, output io.Writer, composeFile string, opts DeployOptions) (deploymentError error) {
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

	targetHost := command.NewHostFromDestination(opts.TargetHost)
	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry == nil {
			if err := term.PrintHeader(output, "Transfer images"); err != nil {
				return err
			}
			if err := docker.TransferImagesViaPipe(ctx, output, composeFile, sourceHost, targetHost); err != nil {
				return err
			}
		} else {
			if err := term.PrintHeader(output, "Run registry"); err != nil {
				return err
			}
			if err := docker.EnsureRegistryRunning(ctx, output, DefaultRegistryContainerName, opts.Registry.Port); err != nil {
				return err
			}

			if err := term.PrintHeader(output, "Open registry SSH tunnel"); err != nil {
				return err
			}
			tunnel, err := ssh.OpenSSHTunnel(ctx, output, opts.TargetHost, opts.Registry.Port, opts.Registry.UseControlSockets)
			if err != nil {
				return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with `--registry-port`", err, opts.Registry.Port)
			}
			defer func() {
				deploymentError = errors.Join(deploymentError, closeTunnel(output, tunnel))
			}()

			if !opts.TargetHost.IsLocalhost() && !opts.Registry.SkipRemotePortCheck {
				if err := term.PrintHeader(output, "Check registry tunnel is not exposed on remote network"); err != nil {
					return err
				}
				if err := docker.CheckTunnelExposure(ctx, output, opts.TargetHost, opts.Registry.Port); err != nil {
					return err
				}
			}

			if err := term.PrintHeader(output, "Transfer via registry"); err != nil {
				return err
			}
			if err := docker.TransferImagesViaRegistry(ctx, output, composeFile, sourceHost, targetHost, opts.Registry.Port); err != nil {
				return err
			}
		}
	}

	if err := term.PrintHeader(output, "Start services"); err != nil {
		return err
	}
	if err := docker.StartServices(ctx, output, composeFile, targetHost, opts.RecreateMode); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Deployment Success"); err != nil {
		return err
	}
	return post_deploy.PrintDeploySuccess(output, composeFile, post_deploy.DefaultMessage(composeFile))
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
