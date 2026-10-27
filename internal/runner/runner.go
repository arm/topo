package runner

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/ssh"
)

// ErrTimeout is returned when a command fails due to context cancellation or deadline.
var ErrTimeout = errors.New("timed out")

type CommandRunner interface {
	Run(context.Context, string) (stdout, stderr string, err error)
}

type Runner interface {
	CommandRunner
	RunWithStdin(ctx context.Context, command string, stdin []byte) (stdout, stderr string, err error)
	BinaryExists(ctx context.Context, bin string) error
}

func For(dest ssh.Destination) Runner {
	if dest.IsPlainLocalhost() {
		return NewLocal()
	}
	return NewSSH(dest)
}
