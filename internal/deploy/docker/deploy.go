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

func Deploy(ctx context.Context, output io.Writer, scope project.Scope, opts DeployOptions) error {
	sourceHost := LocalHost
	commandOutput := term.NewCommandOutput(output)

	buildOutput := term.NewSectionPrinter(commandOutput, "Build images")
	if err := BuildImages(ctx, buildOutput, sourceHost, scope); err != nil {
		return err
	}

	pullOutput := term.NewSectionPrinter(commandOutput, "Pull images")
	if err := PullImages(ctx, pullOutput, sourceHost, scope); err != nil {
		return err
	}

	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry == nil {
			targetHost := NewHostFromDestination(opts.TargetHost)
			if err := transferImagesViaPipe(ctx, commandOutput, sourceHost, targetHost, scope); err != nil {
				return err
			}
		} else {
			if err := transferImagesViaRegistry(ctx, commandOutput, sourceHost, opts.TargetHost, scope, *opts.Registry); err != nil {
				return err
			}
		}
	}

	startOutput := term.NewSectionPrinter(commandOutput, "Start services")
	if err := StartServices(ctx, startOutput, NewHostFromDestination(opts.TargetHost), scope, opts.RecreateMode); err != nil {
		return err
	}

	successOutput := term.NewSectionPrinter(commandOutput, "Deployment Success")
	if err := post_deploy.PrintDeploySuccess(successOutput, scope, post_deploy.DefaultMessage(scope.ComposeFile)); err != nil {
		return err
	}
	return commandOutput.Finish()
}

func transferImagesViaPipe(ctx context.Context, output *term.CommandOutput, sourceHost, targetHost Host, scope project.Scope) error {
	section := term.NewSectionPrinter(output, "Transfer images")
	return TransferImagesViaPipe(ctx, section, sourceHost, targetHost, scope)
}

func transferImagesViaRegistry(ctx context.Context, output *term.CommandOutput, sourceHost Host, targetHost ssh.Destination, scope project.Scope, opts RegistryConfig) (transferErr error) {
	registryOutput := term.NewSectionPrinter(output, "Run registry")
	registryContainerName := opts.ContainerName
	if registryContainerName == "" {
		registryContainerName = DefaultRegistryContainerName
	}
	if err := EnsureRegistryRunning(ctx, registryOutput, registryContainerName, opts.Port); err != nil {
		return err
	}

	tunnelOutput := term.NewSectionPrinter(output, "Open registry SSH tunnel")
	tunnel, err := ssh.OpenTunnel(ctx, tunnelOutput, targetHost, opts.Port)
	if err != nil {
		return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with `--registry-port`", err, opts.Port)
	}
	defer func() {
		transferErr = errors.Join(transferErr, closeTunnel(output, tunnel))
	}()

	if !targetHost.IsLocalhost() && !opts.SkipRemotePortCheck {
		checkOutput := term.NewSectionPrinter(output, "Check registry tunnel is not exposed on remote network")
		if err := deploy.CheckTunnelExposure(ctx, checkOutput, targetHost, opts.Port); err != nil {
			return err
		}
	}

	transferOutput := term.NewSectionPrinter(output, "Transfer via registry")
	if err := TransferImagesViaRegistry(ctx, transferOutput, sourceHost, NewHostFromDestination(targetHost), scope, opts.Port); err != nil {
		return err
	}

	return nil
}

func closeTunnel(output *term.CommandOutput, tunnel *ssh.Tunnel) error {
	ctx, cancel := context.WithTimeout(context.Background(), tunnelCleanupTimeout)
	defer cancel()

	section := term.NewSectionPrinter(output, "Close registry SSH tunnel")
	closeError := tunnel.Close(ctx, section)
	if closeError != nil {
		closeError = fmt.Errorf("failed to close SSH tunnel: %w", closeError)
	}
	return closeError
}
