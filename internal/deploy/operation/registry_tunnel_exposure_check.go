package operation

import (
	"fmt"
	"io"

	"github.com/arm/topo/internal/netprobe"
	"github.com/arm/topo/internal/ssh"
)

type RegistryTunnelExposureCheck struct {
	TargetDest ssh.Destination
	Port       string
}

// NewRegistryTunnelExposureCheck checks whether the registry tunnel port exposes
// the registry to the target's network, rather than being limited to target
// loopback. This can happen when the SSH server permits non-loopback remote
// forwards.
func NewRegistryTunnelExposureCheck(targetDest ssh.Destination, port string) *RegistryTunnelExposureCheck {
	return &RegistryTunnelExposureCheck{TargetDest: targetDest, Port: port}
}

func (c *RegistryTunnelExposureCheck) Description() string {
	return "Check registry tunnel is not exposed on remote network"
}

func (c *RegistryTunnelExposureCheck) Run(w io.Writer) error {
	if c.TargetDest.IsLocalhost() {
		return nil
	}

	host, err := ssh.ResolveHostName(c.TargetDest)
	if err != nil {
		return remotePortCheckErrorWithSuggestion(err, c.Port)
	}
	listening, err := netprobe.IsRemotePortListening(host, c.Port)
	if err != nil {
		return remotePortCheckErrorWithSuggestion(err, c.Port)
	}
	if listening {
		return fmt.Errorf("the remote SSH server is exposing forwarded registry port %s beyond remote loopback; configure the SSH server to bind remote forwards to loopback only, or use `--skip-remote-port-check` if you understand that the registry may be reachable without SSH authentication", c.Port)
	}
	_, _ = fmt.Fprintf(w, "Registry port %s is bound to remote loopback only\n", c.Port)
	return nil
}

func remotePortCheckErrorWithSuggestion(err error, port string) error {
	return fmt.Errorf("cannot conclusively rule out network access to registry port %s because the exposure check did not complete: %w; retry after resolving the connectivity issue, or use `--skip-remote-port-check` if you understand the security risk", port, err)
}
