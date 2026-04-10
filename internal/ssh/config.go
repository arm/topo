package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Config struct {
	HostName string
	User     string
}

func NewConfig(dest Destination) Config {
	output, err := readConfig(dest)
	if err != nil {
		return Config{}
	}
	return NewConfigFromBytes(output)
}

func NewConfigFromBytes(data []byte) Config {
	var config Config
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "hostname":
			config.HostName = fields[1]
		case "user":
			config.User = fields[1]
		}
	}
	return config
}

// IsDestinationAlreadyConfiguredWithAnotherUser checks if the destination is already configured with another user.
// If it is, an error is returned. Otherwise, the user associated with the destination in the SSH config is returned.
func IsDestinationAlreadyConfiguredWithAnotherUser(dest Destination) (string, error) {
	output, err := readConfig(Destination{Host: dest.Host, Port: dest.Port})
	if err != nil {
		return "", err
	}

	hostConfig := NewConfigFromBytes(output)
	isExplicitHostConfig := IsExplicitHostConfig(dest.Host, output)

	if isExplicitHostConfig {
		if dest.User != "" && hostConfig.User != dest.User {
			return "", fmt.Errorf(
				"ssh host/alias %q is already associated with user %q",
				dest.Host,
				hostConfig.User,
			)
		}
	} else if !isExplicitHostConfig && !dest.IsIPLiteral() {
		return "", fmt.Errorf("no explicit host config found for %s", dest.Host)
	} else if !isExplicitHostConfig && dest.IsIPLiteral() {
		if dest.User != "" {
			return dest.User, nil
		}
		return hostConfig.User, nil
	}

	return hostConfig.User, nil
}

func IsExplicitHostConfig(host string, config []byte) bool {
	const marker = ": Applying options for "

	scanner := bufio.NewScanner(bytes.NewReader(config))
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.Contains(line, marker) {
			continue
		}

		hostCandidates := strings.FieldsFunc(strings.TrimSpace(line[strings.Index(line, marker)+len(marker):]), func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t'
		})

		for _, hostCandidate := range hostCandidates {
			if hostCandidate == "" || strings.HasPrefix(hostCandidate, "!") || strings.ContainsAny(hostCandidate, "*?") {
				continue
			}
			if strings.EqualFold(hostCandidate, host) {
				return true
			}
		}
	}

	return false
}

func readConfig(dest Destination) ([]byte, error) {
	return exec.Command("ssh", "-v", "-G", dest.String()).CombinedOutput()
}
