package ssh

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

type sshConfigValues struct {
	host string
	user string
	port string
}

func resolveSSHConfigHost(raw string) string {
	if raw == "" || isExplicitSSHHost(raw) {
		return raw
	}

	output, err := exec.Command("ssh", "-G", raw).Output()
	if err != nil {
		return raw
	}

	values := parseSSHConfigOutput(output)
	return buildResolvedHost(raw, values)
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

func parseSSHConfigOutput(output []byte) sshConfigValues {
	var values sshConfigValues
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "hostname":
			values.host = fields[1]
		case "user":
			values.user = fields[1]
		case "port":
			values.port = fields[1]
		}
	}
	return values
}

func buildResolvedHost(raw string, values sshConfigValues) string {
	hostPart := values.host
	if hostPart == "" {
		hostPart = raw
	}

	if strings.Contains(hostPart, ":") {
		hostPart = "[" + hostPart + "]"
	}

	if values.port != "" {
		hostPart = hostPart + ":" + values.port
	}

	if values.user == "" {
		return hostPart
	}
	return values.user + "@" + hostPart
}
