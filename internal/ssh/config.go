package ssh

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

type Config struct {
	host string
	user string
	port string
}

func NewConfig(destination string) Config {
	output, err := exec.Command("ssh", "-G", destination).Output()
	if err != nil {
		return Config{}
	}
	return NewConfigFromBytes(output)
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

func resolveSSHHost(raw string) (string, string, string) {
	if raw == "" || isExplicitSSHHost(raw) {
		return SplitUserHostPort(raw)
	}

	config := NewConfig(raw)
	return config.user, config.host, config.port
}

func isExplicitSSHHost(raw string) bool {
	if strings.HasPrefix(raw, "ssh://") {
		return true
	}
	if strings.Contains(raw, "@") || strings.Contains(raw, ":") {
		return true
	}
	if strings.HasPrefix(raw, "[") {
		return true
	}
	return false
}
