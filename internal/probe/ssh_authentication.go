package probe

import (
	"context"

	"github.com/arm/topo/internal/runner"
)

// SSHAuthentication verifies SSH connectivity by attempting public key authentication.
func SSHAuthentication(ctx context.Context, r *runner.SSH, acceptNewHostKeys bool) error {
	_, _, err := r.RunWithArgs(ctx, "true", sshAuthArgs(acceptNewHostKeys)...)
	return err
}

func sshAuthArgs(acceptNewHostKeys bool) []string {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PasswordAuthentication=no",
		"-o", "NumberOfPasswordPrompts=0",
	}
	if acceptNewHostKeys {
		args = append(args, "-o", "StrictHostKeyChecking=accept-new")
	} else {
		args = append(args, "-o", "StrictHostKeyChecking=yes")
	}
	return args
}
