package podman

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
	"golang.org/x/sync/errgroup"
)

type DeployOptions struct {
	TargetHost ssh.Destination
}

func Deploy(ctx context.Context, output io.Writer, composeFile string, options DeployOptions) (deployErr error) {
	if err := buildAndPullImages(ctx, output, LocalSocket, composeFile); err != nil {
		return err
	}

	targetSocket := LocalSocket
	var tunnel *ssh.TCPToUnixSocketTunnel
	if !options.TargetHost.IsPlainLocalhost() {
		remoteSocketPath, err := remotePodmanSocketPath(ctx, options.TargetHost)
		if err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("discovered remote Podman socket path: %s", remoteSocketPath))
		if err := term.PrintHeader(output, "Open Podman socket SSH tunnel"); err != nil {
			return err
		}
		tunnel, err = ssh.OpenTCPToUnixSocketTunnel(ctx, output, options.TargetHost, remoteSocketPath)
		if err != nil {
			return fmt.Errorf("failed to open remote Podman socket tunnel: %w", err)
		}
		defer func() {
			if tunnel != nil {
				deployErr = errors.Join(deployErr, closeRemoteTunnel(tunnel))
			}
		}()

		targetSocket = NewSocket(tunnel.SocketURL())
		if err := term.PrintHeader(output, "Transfer images"); err != nil {
			return err
		}
		if err := transferImages(ctx, output, targetSocket, composeFile); err != nil {
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
	return post_deploy.PrintDeploySuccess(output, composeFile, post_deploy.DefaultMessage(composeFile))
}

func buildAndPullImages(ctx context.Context, output io.Writer, socket Socket, composeFile string) error {
	if err := term.PrintHeader(output, "Build images"); err != nil {
		return err
	}
	if err := BuildImages(ctx, output, socket, composeFile); err != nil {
		return err
	}
	if err := term.PrintHeader(output, "Pull images"); err != nil {
		return err
	}
	return PullImages(ctx, output, socket, composeFile)
}

func transferImages(ctx context.Context, output io.Writer, socket Socket, composeFile string) error {
	images, err := compose.ImageNames(composeFile)
	if err != nil {
		return err
	}
	for _, image := range images {
		if err := transferImage(ctx, output, socket, image); err != nil {
			return err
		}
	}
	return nil
}

func transferImage(ctx context.Context, output io.Writer, socket Socket, image string) error {
	pipeReader, pipeWriter := io.Pipe()
	saveCommand := Command(ctx, LocalSocket, "save", image)
	loadCommand := Command(ctx, socket, "load")
	saveCommand.Stdout = pipeWriter
	saveCommand.Stderr = output
	loadCommand.Stdin = pipeReader
	loadCommand.Stdout = output
	loadCommand.Stderr = output

	var group errgroup.Group
	group.Go(func() error {
		err := saveCommand.Run()
		_ = pipeWriter.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to save Podman image %s: %w", image, err)
		}
		return nil
	})
	group.Go(func() error {
		err := loadCommand.Run()
		_ = pipeReader.CloseWithError(err)
		if err != nil {
			return fmt.Errorf("failed to load Podman image %s: %w", image, err)
		}
		return nil
	})
	return group.Wait()
}

func remotePodmanSocketPath(ctx context.Context, target ssh.Destination) (string, error) {
	output, _, err := ssh.RunCommand(ctx, target, "podman info --format '{{.Host.RemoteSocket.Path}}'", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get remote Podman socket path: %w", err)
	}

	socketPath := strings.TrimPrefix(strings.TrimSpace(output), "unix://")
	if !filepath.IsAbs(socketPath) {
		return "", fmt.Errorf("remote Podman reported a non-local API socket path %q", socketPath)
	}
	return socketPath, nil
}

func closeRemoteTunnel(tunnel *ssh.TCPToUnixSocketTunnel) error {
	if err := tunnel.Close(); err != nil {
		return fmt.Errorf("failed to close remote Podman socket tunnel: %w", err)
	}
	return nil
}
