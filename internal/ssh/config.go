package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HostName       string
	User           string
	connectTimeout time.Duration
}

func readConfig(dest Destination) ([]byte, error) {
	return exec.Command("ssh", "-v", "-G", dest.String()).CombinedOutput()
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
		case "connecttimeout":
			if secs, err := strconv.Atoi(fields[1]); err == nil {
				config.connectTimeout = time.Duration(secs) * time.Second
			}
		}
	}
	return config
}

// ConnectTimeout returns the user's configured ConnectTimeout if set, otherwise the fallback.
func (c Config) ConnectTimeout(fallback time.Duration) time.Duration {
	if c.connectTimeout > 0 {
		return c.connectTimeout
	}
	return fallback
}

func IsDestinationAlreadyConfiguredWithAnotherUser(dest Destination) error {
	hostConfig, err := LookupExplicitHostConfig(dest.Host, dest.Port)
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

	if !isExplicitHostConfig(host, output) {
		return Config{}, fmt.Errorf("no explicit host config found for %s", host)
	}

	return NewConfigFromBytes(output), nil
}

func isExplicitHostConfig(host string, config []byte) bool {
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
