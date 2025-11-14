package host

import "fmt"

type Host string

const Local = Host("")

func New(targetHost string) Host {
	return Host(targetHost)
}

func NewSSH(sshTarget string) Host {
	return Host(fmt.Sprintf("ssh://%s", sshTarget))
}
