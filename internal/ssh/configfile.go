package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	sshconfig "github.com/kevinburke/ssh_config"
)

type ConfigDirective = sshconfig.KV

func NewConfigDirectivePath(key, path string) ConfigDirective {
	return ConfigDirective{
		Key:   key,
		Value: filepath.ToSlash(path),
	}
}

func NewConfigDirective(key, value string) ConfigDirective {
	return ConfigDirective{
		Key:   key,
		Value: value,
	}
}

func readConfigFile(path string) (*sshconfig.Config, error) {
	var cfgFile io.Reader
	cfgFile, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfgFile = strings.NewReader("")
		} else {
			return nil, fmt.Errorf("failed to open topo ssh config file: %w", err)

		}
	}

	cfg, err := sshconfig.Decode(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode topo ssh config file: %w", err)
	}

	return cfg, nil
}

func findOrCreateHostBlock(cfg *sshconfig.Config, alias string) (*sshconfig.Host, error) {
	for _, host := range cfg.Hosts {
		hostStr := host.String()
		if hostStr == alias || (host.Matches(alias) && hostStr != "") {
			return host, nil
		}
	}

	pattern, err := sshconfig.NewPattern(alias)
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern for alias %s: %w", alias, err)
	}

	newHost := &sshconfig.Host{
		Patterns: []*sshconfig.Pattern{pattern},
	}

	cfg.Hosts = append(cfg.Hosts, newHost)
	return newHost, nil
}

func addOrReplaceDirective(host *sshconfig.Host, directive ConfigDirective) {
	for i, node := range host.Nodes {
		if kv, ok := node.(*sshconfig.KV); ok && kv.Key == directive.Key {
			host.Nodes[i] = &directive
			return
		}

		if include, ok := node.(*sshconfig.Include); ok && include.String() == directive.String() {
			return
		}
	}

	host.Nodes = append(host.Nodes, &sshconfig.KV{
		Key:   directive.Key,
		Value: directive.Value,
	})
}

func updateConfigFile(path string, host string, directives []ConfigDirective) error {
	cfg, err := readConfigFile(path)
	if err != nil {
		return err
	}

	hostBlock, err := findOrCreateHostBlock(cfg, host)
	if err != nil {
		return fmt.Errorf("failed to find or create host block: %w", err)
	}

	for _, directive := range directives {
		addOrReplaceDirective(hostBlock, directive)
	}

	cfgBytes, err := cfg.MarshalText()
	if err != nil {
		return fmt.Errorf("failed to marshal ssh config: %w", err)
	}

	if err := os.WriteFile(path, cfgBytes, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func CreateOrModifyConfigFile(dest Destination, directives []ConfigDirective) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory for SSH config: %w", err)
	}

	topoConfigPath := filepath.Join(home, ".ssh", "topo_config")
	if err := updateConfigFile(topoConfigPath, dest.Host, directives); err != nil {
		return err
	}

	defaultConfigPath := filepath.Join(home, ".ssh", "config")
	return updateConfigFile(defaultConfigPath, "", []ConfigDirective{
		NewConfigDirectivePath("Include", topoConfigPath),
	})

}

func CheckForLegacyConfigEntries() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory for SSH config: %w", err)
	}

	topoConfigPath := filepath.Join(home, ".ssh", "topo_config")
	info, err := os.Stat(topoConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to check for legacy topo ssh config file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("legacy topo ssh config found at %s; please delete the directory and the corresponding Include directive from your ssh config to proceed. note: you will need to re-run topo setup-keys for any affected targets", topoConfigPath)
	}

	return nil
}
