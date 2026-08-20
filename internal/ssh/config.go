package ssh

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Config struct {
	HostName string
	User     string
	Port     string
}

func LoadConfig(dest Destination) (Config, error) {
	output, err := queryConfig(dest)
	if err != nil {
		return Config{}, fmt.Errorf("failed to query SSH config for '%s': %w", dest.String(), err)
	}
	return ParseConfigFromBytes(output), nil
}

func ResolveHostName(dest Destination) (string, error) {
	if dest.IsPlainLocalhost() {
		return dest.Host, nil
	}

	config, err := LoadConfig(dest)
	if err != nil {
		return "", err
	}
	return config.HostName, nil
}

func ParseConfigFromBytes(data []byte) Config {
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
		case "port":
			config.Port = fields[1]
		}
	}
	return config
}

func (c Config) AsKnownHostsEntry() string {
	if c.Port == "" || c.Port == "22" {
		return c.HostName
	}

	return fmt.Sprintf("[%s]:%s", c.HostName, c.Port)
}

func GetUserFromConfig(dest Destination) (string, error) {
	output, err := queryConfig(Destination{Host: dest.Host, Port: dest.Port})
	if err != nil {
		return "", err
	}
	return ResolveConfiguredUser(dest, output)
}

func ResolveConfiguredUser(dest Destination, configOutput []byte) (string, error) {
	hostConfig := ParseConfigFromBytes(configOutput)

	if IsExplicitHostConfig(dest.Host, configOutput) {
		if hostConfig.User != "" && dest.User != "" && hostConfig.User != dest.User {
			return "", fmt.Errorf(
				"ssh host/alias %q is already associated with user %q",
				dest.Host,
				hostConfig.User,
			)
		}
		if dest.User != "" {
			return dest.User, nil
		}
		return hostConfig.User, nil
	}

	if dest.User != "" {
		return dest.User, nil
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

func queryConfig(dest Destination) ([]byte, error) {
	return queryConfigContext(context.Background(), dest)
}

func queryConfigContext(ctx context.Context, dest Destination) ([]byte, error) {
	return exec.CommandContext(ctx, "ssh", "-v", "-G", dest.String()).CombinedOutput()
}
