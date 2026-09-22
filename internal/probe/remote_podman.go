package probe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/ssh"
)

var (
	ErrRemotePodmanSocketResolutionFailed  = errors.New("remote Podman socket resolution failed")
	ErrRemotePodmanForwardingFailed        = errors.New("remote Podman forwarding failed")
	ErrRemotePodmanAPIRequestFailed        = errors.New("remote Podman API request failed")
	ErrRemotePodmanSocketTunnelCloseFailed = errors.New("failed to close remote Podman socket tunnel")
)

type RemotePodmanProbeResult struct {
	SocketPath string
	Err        error
}

func CheckRemotePodmanAPI(ctx context.Context, target ssh.Destination) RemotePodmanProbeResult {
	remoteSocketPath, tunnel, result := openRemotePodmanTunnel(ctx, target)
	if result.Err != nil {
		return result
	}

	probeErr := runCommand(podman.Command(ctx, podman.NewSocket(tunnel.SocketURL()), "info"))
	if probeErr == nil {
		composeCommand, err := podman.ComposeSocketRawCommand(ctx, podman.NewSocket(tunnel.SocketURL()), "ls")
		if err != nil {
			probeErr = err
		} else {
			probeErr = runCommand(composeCommand)
		}
	}
	closeErr := tunnel.Close()
	if probeErr != nil {
		return RemotePodmanProbeResult{
			SocketPath: remoteSocketPath,
			Err:        fmt.Errorf("%w: %w", ErrRemotePodmanAPIRequestFailed, probeErr),
		}
	}
	if closeErr != nil {
		return RemotePodmanProbeResult{
			SocketPath: remoteSocketPath,
			Err:        fmt.Errorf("%w: %w", ErrRemotePodmanSocketTunnelCloseFailed, closeErr),
		}
	}
	return RemotePodmanProbeResult{SocketPath: remoteSocketPath}
}

func openRemotePodmanTunnel(ctx context.Context, target ssh.Destination) (string, *ssh.TCPToUnixSocketTunnel, RemotePodmanProbeResult) {
	remoteSocketPath, err := podman.ResolveRemoteSocketPath(ctx, target)
	if err != nil {
		return "", nil, RemotePodmanProbeResult{
			Err: fmt.Errorf("%w: %w", ErrRemotePodmanSocketResolutionFailed, err),
		}
	}
	tunnel, err := podman.OpenRemoteSocketTunnel(ctx, io.Discard, target, remoteSocketPath)
	if err != nil {
		return "", nil, RemotePodmanProbeResult{
			SocketPath: remoteSocketPath,
			Err:        fmt.Errorf("%w: %w", ErrRemotePodmanForwardingFailed, err),
		}
	}
	return remoteSocketPath, tunnel, RemotePodmanProbeResult{}
}

func CheckPodmanComposeProvider(ctx context.Context) error {
	return runCommand(podman.ComposeRawCommand(ctx, "version"))
}

func runCommand(cmd *exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		return command.NewError(cmd, err)
	}
	return nil
}
