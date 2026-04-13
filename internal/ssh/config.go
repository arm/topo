package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrNoExplicitHostConfig = errors.New("no explicit host config found")

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

func IsDestinationAlreadyConfiguredWithAnotherUser(dest Destination) error {
	hostConfig, err := LookupExplicitHostConfig(dest.Host, dest.Port)
	if errors.Is(err, ErrNoExplicitHostConfig) {
		return nil
	}
	if err != nil {
		return err
	}

	if dest.User != "" && hostConfig.User != dest.User {
		return fmt.Errorf("ssh host/alias %q is already associated with user %q", dest.Host, hostConfig.User)
	}
	return nil
}

// LookupExplicitHostConfig returns configuration, only if there's an explicit entry for the given host/port
func LookupExplicitHostConfig(host, port string) (Config, error) {
	dest := Destination{Host: host, Port: port}
	output, err := readConfig(dest)
	if err != nil {
		return Config{}, err
	}

	if !IsExplicitHostConfig(host, output) {
		return Config{}, fmt.Errorf("%w: %s", ErrNoExplicitHostConfig, host)
	}

	return NewConfigFromBytes(output), nil
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
