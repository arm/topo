package ssh

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SSHAlias struct {
	Alias string
	Host  string
	User  string
	Port  string
}

var PlainLocalhostAlias = SSHAlias{Alias: "localhost"}

func (h SSHAlias) ControlSocketPath() string {
	hash := sha256.Sum256([]byte(h.Alias))
	hostHash := fmt.Sprintf("%x", hash[:8]) // Hash to avoid filepath limits
	return filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%s", hostHash))
}

func (h SSHAlias) FormatSSHConnectCommand(useControlSockets bool, registryPort string) []string {
	args := []string{"ssh", "-N", "-o", "ExitOnForwardFailure=yes"}
	if h.Port != "22" && h.Port != "" {
		args = append(args, "-p", h.Port)
	}
	if useControlSockets {
		args = append(args,
			"-fMS", h.ControlSocketPath(),
		)
	}

	args = append(args, "-R", fmt.Sprintf("%s:127.0.0.1:%s", registryPort, registryPort), h.Alias)
	return args
}

func (h SSHAlias) FormatSSHExitCommand() []string {
	args := []string{"ssh"}
	if h.Port != "22" && h.Port != "" {
		args = append(args, "-p", h.Port)
	}
	args = append(args, "-S", h.ControlSocketPath(), "-O", "exit", h.Alias)
	return args
}

func (h SSHAlias) rawAddress() string {
	if h.Alias != "" {
		return h.Alias
	}
	if h.Host == "" {
		return ""
	}
	host := h.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if h.User != "" {
		host = fmt.Sprintf("%s@%s", h.User, host)
	}
	if h.Port != "" {
		host = fmt.Sprintf("%s:%s", host, h.Port)
	}
	return host
}

func (h SSHAlias) AsURI() string {
	return "ssh://" + h.rawAddress()
}

func (sa SSHAlias) GetHost() string {
	return sa.Host
}

func (h SSHAlias) IsLocalhost() bool {
	return strings.EqualFold(h.Host, "localhost") || h.Host == "127.0.0.1"
}

func NewSSHAlias(raw string) SSHAlias {
	output, err := exec.Command("ssh", "-G", raw).Output()
	if err != nil {
		return SSHAlias{}
	}
	user, host, port := parseSSHConfigOutput(output)
	return SSHAlias{User: user, Host: host, Port: port, Alias: raw}
}

func IsAlias(raw string) bool {
	if strings.HasPrefix(raw, "ssh://") {
		return false
	}
	if strings.Contains(raw, "@") || strings.Contains(raw, ":") {
		return false
	}
	if strings.HasPrefix(raw, "[") {
		return false
	}
	return true
}

func parseSSHConfigOutput(output []byte) (string, string, string) {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var user, host, port string
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "hostname":
			host = fields[1]
		case "user":
			user = fields[1]
		case "port":
			port = fields[1]
		}
	}
	return user, host, port
}
