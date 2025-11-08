package host

type Host string

const Local = Host("")

func NewSSH(sshTarget string) Host {
	return Host(sshTarget)
}

func (h Host) DockerCommandArgs() []string {
	if h == "" {
		return nil
	}
	return []string{"-H", string(h)}
}
