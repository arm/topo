package host

type Host string

const Local = Host("")

func New(targetHost string) Host {
	return Host(targetHost)
}

func (h Host) DockerCommandArgs() []string {
	if h == "" {
		return nil
	}
	return []string{"-H", string(h)}
}
