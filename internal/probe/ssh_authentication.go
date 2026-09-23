package probe

import (
	"context"

	"github.com/arm/topo/internal/runner"
)

// SSHAuthentication verifies SSH connectivity by attempting public key authentication.
func SSHAuthentication(ctx context.Context, r *runner.SSH) error {
	_, _, err := r.RunWithArgs(ctx, "true",
		"-o", "BatchMode=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PasswordAuthentication=no",
		"-o", "NumberOfPasswordPrompts=0",
		"-o", "StrictHostKeyChecking=yes",
	)
	return err
}
