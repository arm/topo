package podman

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/ssh"
)

type Socket struct {
	url string
}

var LocalSocket Socket

func NewSocket(url string) Socket {
	return Socket{url: url}
}

func (socket Socket) ConfigurePodmanEnv(environment []string) []string {
	if socket.url == "" {
		return environment
	}
	return append(environment, "CONTAINER_HOST="+socket.url)
}

// ResolveRemoteSocketPath returns the absolute Unix socket path reported by Podman
// on target.
func ResolveRemoteSocketPath(ctx context.Context, target ssh.Destination) (string, error) {
	output, _, err := ssh.RunCommand(ctx, target, "podman info --format '{{.Host.RemoteSocket.Path}}'", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get remote Podman socket path: %w", err)
	}

	socketPath := strings.TrimPrefix(strings.TrimSpace(output), "unix://")
	if !path.IsAbs(socketPath) {
		return "", fmt.Errorf("remote Podman reported a non-local API socket path %q", socketPath)
	}
	return socketPath, nil
}

// TunnelRemoteSocketPath resolves remote socket path on the target and tunnels it to a tcp endpoint on the host.
func TunnelRemoteSocketPath(ctx context.Context, w io.Writer, target ssh.Destination) (*ssh.TCPToUnixSocketTunnel, error) {
	remoteSocketPath, err := ResolveRemoteSocketPath(ctx, target)
	if err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("discovered remote Podman socket path: %s", remoteSocketPath))
	tunnel, err := ssh.OpenTCPToUnixSocketTunnel(ctx, w, target, remoteSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote Podman socket tunnel: %w", err)
	}
	return tunnel, nil
}
