package runner

import "github.com/arm/topo/internal/ssh"

type Runner interface {
	Run(command string) (string, error)
	RunWithStdin(command string, stdin []byte) (string, error)
}

func For(dest ssh.Destination, opts SSHOptions) Runner {
	if dest.IsPlainLocalhost() {
		return NewLocal()
	}
	return NewSSH(dest, opts)
}
