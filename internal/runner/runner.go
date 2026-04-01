package runner

import "github.com/arm/topo/internal/ssh"

// Runner is the common interface implemented by both SSH and Local.
type Runner interface {
	Run(command string) (string, error)
	RunWithStdin(command string, stdin []byte) (string, error)
}

// For returns an SSH runner for remote destinations and a Local runner for localhost.
func For(dest ssh.Destination, opts SSHOptions) Runner {
	if dest.IsPlainLocalhost() {
		local := NewLocal()
		return &local
	}
	r := NewSSH(dest, opts)
	return &r
}
