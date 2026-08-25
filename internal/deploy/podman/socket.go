package podman

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/arm/topo/internal/ssh"
)

type Socket struct {
	url string
}

var LocalSocket Socket

func NewSocket(url string) Socket {
	return Socket{url: url}
}

// NewRemoteSocket returns a Podman SSH connection to the target's API socket.
func NewRemoteSocket(ctx context.Context, target ssh.Destination) (Socket, error) {
	remoteSocketPath, err := ResolveRemoteSocketPath(ctx, target)
	if err != nil {
		return Socket{}, err
	}
	return NewSocket(target.String() + remoteSocketPath), nil
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
