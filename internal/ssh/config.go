package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HostName       string
	User           string
	connectTimeout time.Duration
}

type ConfigDirective struct {
	Key   string
	Value string
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

	if !IsExplicitHostConfig(host, output) {
		return Config{}, fmt.Errorf("no explicit host config found for %s", host)
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

func NewDirectiveIdentityFile(path string) ConfigDirective {
	return ConfigDirective{
		Key:   "IdentityFile",
		Value: filepath.ToSlash(path), // needs to be this way even on Windows to work with ssh config parsing, which generally accepts forward slashes
	}
}

func NewDirective(key, value string) ConfigDirective {
	return ConfigDirective{
		Key:   key,
		Value: value,
	}
}

func (d ConfigDirective) String() string {
	return fmt.Sprintf("%s %s", d.Key, d.Value)
}

func CreateConfig(dest Destination, targetSlug string) error {
	return CreateOrModifyConfig(dest, targetSlug, nil)
}

func CreateOrModifyConfig(dest Destination, targetSlug string, directives []ConfigDirective) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory for SSH config: %w", err)
	}

	topoConfigDir := filepath.Join(home, ".ssh", "topo_config")
	topoConfigPath := filepath.Join(topoConfigDir, fmt.Sprintf("topo_%s.conf", targetSlug))

	mainConfigPath := filepath.Join(home, ".ssh", "config")

	// ssh config parsing expects forward slashes (even on Windows)
	includeLine := fmt.Sprintf("Include %s", filepath.ToSlash(topoConfigDir+"/*.conf"))

	if err := os.MkdirAll(topoConfigDir, 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", topoConfigDir, err)
	}

	existingTopoContent, errTopo := getFileContents(topoConfigPath)
	if errTopo != nil {
		return errTopo
	}

	existingMainContent, errMain := getFileContents(mainConfigPath)
	if errMain != nil {
		return errMain
	}

	var fragmentToWrite []byte
	if len(existingTopoContent) == 0 {
		fragmentToWrite = buildConfigFragment(dest, directives)
	} else {
		fragmentToWrite = mergeOwnedConfigDirectives(existingTopoContent, directives)
	}

	if !bytes.Equal(existingTopoContent, fragmentToWrite) {
		if err := os.WriteFile(topoConfigPath, fragmentToWrite, 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", topoConfigPath, err)
		}
	}

	if !hasIncludeLine(existingMainContent, includeLine) {
		updated := slices.Concat([]byte(includeLine+"\n\n"), existingMainContent)
		return os.WriteFile(mainConfigPath, updated, 0o600)
	}

	return nil
}

func getFileContents(filePath string) ([]byte, error) {
	existingContent, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	} else if os.IsNotExist(err) {
		existingContent = []byte{}
	}

	return existingContent, nil
}

func hasIncludeLine(data []byte, includeLine string) bool {
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.EqualFold(trimmed, includeLine) {
			return true
		}
	}
	return false
}

func buildConfigFragment(dest Destination, directives []ConfigDirective) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "Host %s\n", dest.Host)
	if dest.Host != "" && (dest.User != "" || net.ParseIP(dest.Host) != nil) {
		fmt.Fprintf(&b, "  HostName %s\n", dest.Host)
	}
	if dest.User != "" {
		fmt.Fprintf(&b, "  User %s\n", dest.User)
	}
	if dest.Port != "" {
		fmt.Fprintf(&b, "  Port %s\n", dest.Port)
	}

	for _, directive := range directives {
		fmt.Fprintf(&b, "  %s\n", directive.String())
	}
	return []byte(b.String())
}

func mergeOwnedConfigDirectives(existing []byte, directives []ConfigDirective) []byte {
	var merged [][]byte
	directiveKeys := make(map[string]bool)
	for _, d := range directives {
		directiveKeys[d.Key] = true
	}

	for line := range bytes.SplitSeq(bytes.TrimRight(existing, "\n"), []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		key := extractConfigKey(trimmed)

		if directiveKeys[key] {
			continue
		}
		merged = append(merged, line)
	}

	for _, directive := range directives {
		merged = append(merged, []byte("  "+directive.String()))
	}

	return append(bytes.Join(merged, []byte("\n")), '\n')
}

func extractConfigKey(line []byte) string {
	parts := bytes.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	return string(parts[0])
}
