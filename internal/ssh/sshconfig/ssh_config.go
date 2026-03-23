package sshconfig

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arm/topo/internal/ssh"
)

type SSHConfigDirective struct {
	Key   string
	Value string
}

func (d SSHConfigDirective) String() string {
	return fmt.Sprintf("%s %s", d.Key, d.Value)
}

func NewDirectiveIdentityFile(path string) SSHConfigDirective {
	return SSHConfigDirective{
		Key:   "IdentityFile",
		Value: filepath.ToSlash(path), // needs to be this way even on Windows to work with ssh config parsing, which generally accepts forward slashes
	}
}

func NewDirective(key, value string) SSHConfigDirective {
	return SSHConfigDirective{
		Key:   key,
		Value: value,
	}
}

func CreateSSHConfig(targetHost string, targetSlug string) error {
	return CreateOrModifySSHConfig(targetHost, targetSlug, nil)
}

func CreateOrModifySSHConfig(targetHost string, targetSlug string, directives []SSHConfigDirective) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory for SSH config: %w", err)
	}

	sshTopoConfigDir := filepath.Join(home, ".ssh", "topo_config")
	sshTopoConfigPath := filepath.Join(sshTopoConfigDir, fmt.Sprintf("topo_%s.conf", targetSlug))

	mainConfigPath := filepath.Join(home, ".ssh", "config")

	// ssh config parsing expects forward slashes (even on Windows)
	includeLine := fmt.Sprintf("Include %s", filepath.ToSlash(sshTopoConfigDir+"/*.conf"))

	if err := os.MkdirAll(sshTopoConfigDir, 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", sshTopoConfigDir, err)
	}

	existingTopoContent, errTopo := getFileContents(sshTopoConfigPath)
	if errTopo != nil {
		return errTopo
	}

	existingMainContent, errMain := getFileContents(mainConfigPath)
	if errMain != nil {
		return errMain
	}

	var fragmentToWrite []byte
	if len(existingTopoContent) == 0 {
		fragmentToWrite = buildSSHConfigFragment(targetHost, directives)
	} else {
		fragmentToWrite = mergeOwnedSSHConfigDirectives(existingTopoContent, directives)
	}

	if !bytes.Equal(existingTopoContent, fragmentToWrite) {
		if err := os.WriteFile(sshTopoConfigPath, fragmentToWrite, 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", sshTopoConfigPath, err)
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

func buildSSHConfigFragment(targetHost string, directives []SSHConfigDirective) []byte {
	user, host, port := ssh.SplitUserHostPort(targetHost)
	hostAlias := host
	if hostAlias == "" {
		hostAlias = targetHost
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Host %s\n", hostAlias)
	if host != "" && (strings.Contains(targetHost, "@") || strings.Contains(targetHost, ":")) {
		fmt.Fprintf(&b, "  HostName %s\n", host)
	}
	if user != "" {
		fmt.Fprintf(&b, "  User %s\n", user)
	}
	if port != "" {
		fmt.Fprintf(&b, "  Port %s\n", port)
	}

	for _, directive := range directives {
		fmt.Fprintf(&b, "  %s\n", directive.String())
	}
	return []byte(b.String())
}

func mergeOwnedSSHConfigDirectives(existing []byte, directives []SSHConfigDirective) []byte {
	var merged [][]byte
	directiveKeys := make(map[string]bool)
	for _, d := range directives {
		directiveKeys[d.Key] = true
	}

	for line := range bytes.SplitSeq(bytes.TrimRight(existing, "\n"), []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		key := extractSSHConfigKey(trimmed)

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

func extractSSHConfigKey(line []byte) string {
	parts := bytes.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	return string(parts[0])
}
