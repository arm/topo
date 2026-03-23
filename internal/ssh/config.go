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
	HostName       string
	connectTimeout time.Duration
}

func NewConfig(dest Destination) Config {
	output, err := exec.Command("ssh", "-G", dest.String()).Output()
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
			config.HostName = fields[1]
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
