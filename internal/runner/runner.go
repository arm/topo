package runner

import "github.com/arm/topo/internal/ssh"

type Runner interface {
	Run(command string) (string, error)
}

type StdinRunner interface {
	RunWithStdin(command string, stdin []byte) (string, error)
}

// Executor is the common interface implemented by both SSH and Local.
type Executor interface {
	Runner
	StdinRunner
}

// For returns an SSH runner for remote destinations and a Local runner for localhost.
func For(dest ssh.Destination, opts SSHOptions) Executor {
	if dest.IsPlainLocalhost() {
		local := NewLocal()
		return &local
	}
	r := NewSSH(dest, opts)
	return &r
}
