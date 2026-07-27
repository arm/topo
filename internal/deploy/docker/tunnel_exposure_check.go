package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/arm/topo/internal/netprobe"
	"github.com/arm/topo/internal/ssh"
)

// CheckTunnelExposure checks whether the registry tunnel port exposes the
// registry to the target's network, rather than being limited to target
// loopback. This can happen when the SSH server permits non-loopback remote
// forwards.
func CheckTunnelExposure(ctx context.Context, output io.Writer, targetDest ssh.Destination, port string) error {
	listening, err := netprobe.IsRemotePortListening(ctx, targetDest, port)
	if err != nil {
		return fmt.Errorf("cannot conclusively rule out network access to registry port %s because the exposure check did not complete: %w; retry after resolving the connectivity issue, or use `--skip-remote-port-check` if you understand the security risk", port, err)
	}
	if listening {
		return fmt.Errorf("the remote SSH server is exposing forwarded registry port %s beyond remote loopback; configure the SSH server to bind remote forwards to loopback only, or use `--skip-remote-port-check` if you understand that the registry may be reachable without SSH authentication", port)
	}
	_, _ = fmt.Fprintf(output, "Registry port %s is bound to remote loopback only\n", port)
	return nil
}
