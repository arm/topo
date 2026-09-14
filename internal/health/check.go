package health

import (
	"context"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/runner"
)

func CheckOpenSSHAvailable(ctx context.Context, r runner.Runner, sshBinary string) *DependencyCheckFailure {
	_, stderr, err := r.Run(ctx, sshBinary+" -V")
	if err != nil {
		return &DependencyCheckFailure{Message: err.Error()}
	}
	if !strings.Contains(stderr, "OpenSSH_") {
		return &DependencyCheckFailure{
			Message: fmt.Sprintf("%q does not resolve to OpenSSH: %s", sshBinary, stderr),
			Fix: &Fix{
				Description: "Install OpenSSH and ensure its ssh executable is first on PATH",
			},
		}
	}
	return nil
}
