package ssh

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

type SSHConfigValues struct {
	host       Host
	user       string
	port       string
	configName string
}

func resolveSSHConfigHost(raw string) SSHConfigValues {
	if raw == "" || isExplicitSSHHost(raw) {
		user, host, port := SplitUserHostPort(raw)
		return SSHConfigValues{user: user, host: Host(host), port: port, configName: ""}
	}

	output, err := exec.Command("ssh", "-G", raw).Output()
	if err != nil {
		return SSHConfigValues{user: "", host: "", port: "", configName: ""}
	}
	user, host, port := parseSSHConfigOutput(output)
	return SSHConfigValues{user: user, host: Host(host), port: port, configName: raw}
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
