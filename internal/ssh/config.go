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

type Config struct {
	alias string
	host  string
	user  string
	port  string
}

func (c Config) ControlSocketPath() string {
	hash := sha256.Sum256([]byte(c.alias))
	hostHash := fmt.Sprintf("%x", hash[:8]) // Hash to avoid filepath limits
	return filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%s", hostHash))
}

func (c Config) FormatSSHConnectCommand(useControlSockets bool, registryPort string) []string {
	args := []string{"ssh", "-N", "-o", "ExitOnForwardFailure=yes"}
	if c.port != "22" && c.port != "" {
		args = append(args, "-p", c.port)
	}
	if useControlSockets {
		args = append(args,
			"-fMS", c.ControlSocketPath(),
		)
	}

	args = append(args, "-R", fmt.Sprintf("%s:127.0.0.1:%s", registryPort, registryPort), c.alias)
	return args
}

func (c Config) FormatSSHExitCommand() []string {
	args := []string{"ssh"}
	if c.port != "22" && c.port != "" {
		args = append(args, "-p", c.port)
	}
	args = append(args, "-S", c.ControlSocketPath(), "-O", "exit", c.alias)
	return args
}

func (c Config) rawAddress() string {
	if c.alias != "" {
		return c.alias
	}
	if c.host == "" {
		return ""
	}
	host := c.host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if c.user != "" {
		host = fmt.Sprintf("%s@%s", c.user, host)
	}
	if c.port != "" {
		host = fmt.Sprintf("%s:%s", host, c.port)
	}
	return host
}

func (c Config) AsURI() string {
	return "ssh://" + c.rawAddress()
}

func (c Config) GetHost() string {
	return c.host
}

func (c Config) IsLocalhost() bool {
	return strings.EqualFold(c.host, "localhost") || c.host == "127.0.0.1"
}

func NewConfig(raw string) Config {
	output, err := exec.Command("ssh", "-G", raw).Output()
	if err != nil {
		return Config{}
	}
	return NewConfigFromBytes(output)
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

func NewConfigFromBytes(data []byte) Config {
	var config Config
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "hostname":
			config.host = fields[1]
		case "user":
			config.user = fields[1]
		case "port":
			config.port = fields[1]
		}
	}
	return config
}
