package runner

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

// ErrTimeout is returned when a command fails due to context cancellation or deadline.
var ErrTimeout = errors.New("timed out")

type CommandRunner interface {
	Run(ctx context.Context, program string, args ...string) (stdout, stderr string, err error)
	RunShell(ctx context.Context, script string) (stdout, stderr string, err error)
}

type Runner interface {
	CommandRunner
	RunWithStdin(ctx context.Context, stdin []byte, program string, args ...string) (stdout, stderr string, err error)
	BinaryExists(ctx context.Context, bin string) error
}

func For(dest ssh.Destination) Runner {
	if dest.IsPlainLocalhost() {
		return NewLocal()
	}
	return NewSSH(dest)
}

func commandLine(program string, args ...string) string {
	words := slices.Concat([]string{program}, args)
	for i, word := range words {
		words[i] = command.QuoteArg(word)
	}
	return strings.Join(words, " ")
}
