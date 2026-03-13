package ssh

type SSHConnection interface {
	ControlSocketPath() string
	FormatSSHConnectCommand(bool, string) []string
	FormatSSHExitCommand() []string
	GetHost() string
	IsLocalhost() bool
}
