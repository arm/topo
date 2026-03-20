package sshconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/ssh"
)

func ModifySSHConfig(targetHost string, privKeyPath string, targetSlug string, dryRun bool, output io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory for SSH config: %w", err)
	}

	sshTopoConfigDir := filepath.Join(home, ".ssh", "topo_config")
	sshTopoConfigPath := filepath.Join(sshTopoConfigDir, fmt.Sprintf("topo_%s.conf", targetSlug))

	mainConfigPath := filepath.Join(home, ".ssh", "config")

	// ssh config parsing expects forward slashes (even on Windows)
	includeLine := fmt.Sprintf("Include %s", filepath.ToSlash(sshTopoConfigDir+"/*.conf"))

	if dryRun {
		if output == nil {
			return errors.New("dry run requested but no output writer provided for SSH config changes")
		}

		if err := term.PrintHeader(output, "Update local SSH config for key-based authentication"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(output, "Will update %s to include:\n- %s\n", mainConfigPath, includeLine); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(output, "Will create %s and place the target SSH config there pointing at the key that was just created.\n", sshTopoConfigPath); err != nil {
			return err
		}

		return nil
	}

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

	configContent := buildSSHConfigFragment(targetHost, privKeyPath)
	mergedConfigContent := mergeOwnedSSHConfigDirectives(existingTopoContent, configContent, privKeyPath)

	if string(existingTopoContent) != mergedConfigContent {
		if err := os.WriteFile(sshTopoConfigPath, []byte(mergedConfigContent), 0o600); err != nil {
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

func buildSSHConfigFragment(targetHost string, privKeyPath string) string {
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

	// needs to be this way even on Windows to work with ssh config parsing, which generally accepts forward slashes
	fmt.Fprintf(&b, "  IdentityFile %s\n", filepath.ToSlash(privKeyPath))
	b.WriteString("  IdentitiesOnly yes\n")
	return b.String()
}

func mergeOwnedSSHConfigDirectives(existing []byte, fallbackConfigContent string, privKeyPath string) string {
	if len(existing) == 0 {
		return fallbackConfigContent
	}

	identityLine := []byte(fmt.Sprintf("  IdentityFile %s", filepath.ToSlash(privKeyPath)))
	identitiesOnlyLine := []byte("  IdentitiesOnly yes")
	var merged [][]byte

	for line := range bytes.SplitSeq(bytes.TrimRight(existing, "\n"), []byte("\n")) {
		trimmed := bytes.TrimSpace(line)

		switch {
		case bytes.HasPrefix(trimmed, []byte("IdentityFile ")):
			continue
		case bytes.HasPrefix(trimmed, []byte("IdentitiesOnly ")):
			continue
		default:
			merged = append(merged, line)
		}
	}

	merged = append(merged, identityLine, identitiesOnlyLine)

	return string(bytes.Join(merged, []byte("\n"))) + "\n"
}
