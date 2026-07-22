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
	if err := runStep(output, "Build images", func() error {
		return docker.BuildImages(ctx, output, composeFile, sourceHost)
	}); err != nil {
		return err
	}
	if err := runStep(output, "Pull images", func() error {
		return docker.PullImages(ctx, output, composeFile, sourceHost)
	}); err != nil {
		return err
	}

	targetHost := command.NewHostFromDestination(opts.TargetHost)
	if !opts.TargetHost.IsPlainLocalhost() && opts.Registry == nil {
		if err := runStep(output, "Transfer images", func() error {
			return docker.TransferImagesViaPipe(ctx, output, composeFile, sourceHost, targetHost)
		}); err != nil {
			return err
		}
	}
	if !opts.TargetHost.IsPlainLocalhost() && opts.Registry != nil {
		if err := runStep(output, "Run registry", func() error {
			return docker.EnsureRegistryRunning(ctx, output, DefaultRegistryContainerName, opts.Registry.Port)
		}); err != nil {
			return err
		}
		var tunnel *ssh.SSHTunnel
		if err := runStep(output, "Open registry SSH tunnel", func() error {
			var err error
			tunnel, err = ssh.OpenSSHTunnel(ctx, output, opts.TargetHost, opts.Registry.Port, opts.Registry.UseControlSockets)
			if err != nil {
				return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with `--registry-port`", err, opts.Registry.Port)
			}
			return nil
		}); err != nil {
			return err
		}
		defer func() {
			deploymentError = errors.Join(deploymentError, closeTunnel(output, tunnel))
		}()

		if !opts.TargetHost.IsLocalhost() && !opts.Registry.SkipRemotePortCheck {
			if err := runStep(output, "Check registry tunnel is not exposed on remote network", func() error {
				return docker.CheckTunnelExposure(ctx, output, opts.TargetHost, opts.Registry.Port)
			}); err != nil {
				return err
			}
		}
		if err := runStep(output, "Transfer via registry", func() error {
			return docker.TransferImagesViaRegistry(ctx, output, composeFile, sourceHost, targetHost, opts.Registry.Port)
		}); err != nil {
			return err
		}
	}

	if err := runStep(output, "Start services", func() error {
		return docker.StartServices(ctx, output, composeFile, targetHost, opts.RecreateMode)
	}); err != nil {
		return err
	}
	return runStep(output, "Deployment Success", func() error {
		return post_deploy.PrintDeploySuccess(output, composeFile, post_deploy.DefaultMessage(composeFile))
	})
}

func runStep(output io.Writer, description string, run func() error) error {
	if output != nil {
		if err := term.PrintHeader(output, description); err != nil {
			return err
		}
	}
	return run()
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
