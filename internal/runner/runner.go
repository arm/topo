package runner

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/ssh"
)

// ErrTimeout is returned when a command fails due to context cancellation or deadline.
var ErrTimeout = errors.New("command timed out")

type Runner interface {
	Run(ctx context.Context, command string) (string, error)
	RunWithStdin(ctx context.Context, command string, stdin []byte) (string, error)
}

func For(dest ssh.Destination, opts SSHOptions) Runner {
	if dest.IsPlainLocalhost() {
		return NewLocal()
	}
	return NewSSH(dest, opts)
}
