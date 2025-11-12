package host

type Host string

const Local = Host("")

func New(targetHost string) Host {
	return Host(targetHost)
}
