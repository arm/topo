package command

type Host struct {
	Value string
}

func NewHost(value string) Host {
	return Host{Value: value}
}

func NewLocalHost() Host {
	return Host{Value: "ssh://localhost"}
}

func (h Host) IsPlainLocalhost() bool {
	return h.Value == "ssh://localhost"
}
