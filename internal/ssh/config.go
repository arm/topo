package ssh

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HostName            string
	User                string
	connectTimeout      time.Duration
	matchedHostPatterns []string
}

func NewConfig(dest Destination) Config {
	output, err := exec.Command("ssh", "-v", "-G", dest.String()).CombinedOutput()
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
		if matchedPatterns := extractMatchedHostPatterns(line); len(matchedPatterns) > 0 {
			config.matchedHostPatterns = append(config.matchedHostPatterns, matchedPatterns...)
		}

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

func (c Config) HasExactHostMatch(host string) bool {
	for _, pattern := range c.matchedHostPatterns {
		if pattern == "" || strings.HasPrefix(pattern, "!") {
			continue
		}
		if strings.EqualFold(pattern, host) {
			return true
		}
	}
	return false
}

func extractMatchedHostPatterns(line string) []string {
	const marker = ": Applying options for "

	idx := strings.Index(line, marker)
	if idx == -1 {
		return nil
	}

	return strings.FieldsFunc(strings.TrimSpace(line[idx+len(marker):]), func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
}
