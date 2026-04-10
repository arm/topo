package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	sshconfig "github.com/kevinburke/ssh_config"
)

const (
	DefaultConfigFileName = "config"
	TopoConfigFileName    = "topo_config"
)

type ConfigDirectiveModifier interface {
	Apply(host *sshconfig.Host)
}

type EnsureConfigDirective struct {
	sshconfig.KV
}

func NewEnsureConfigDirective(key, value string) EnsureConfigDirective {
	return EnsureConfigDirective{
		KV: sshconfig.KV{
			Key:   key,
			Value: value,
		},
	}
}

func NewEnsureConfigDirectivePath(key, path string) EnsureConfigDirective {
	return EnsureConfigDirective{
		KV: sshconfig.KV{
			Key:   key,
			Value: filepath.ToSlash(path),
		},
	}
}

func (d EnsureConfigDirective) Apply(host *sshconfig.Host) {
	for i, node := range host.Nodes {
		if directiveMatches(node, d.KV) {
			host.Nodes[i] = &d
			return
		}
	}

	host.Nodes = append(host.Nodes, &d)
}

type RemoveConfigDirective struct {
	sshconfig.KV
}

func NewRemoveConfigDirectivePath(key, value string) RemoveConfigDirective {
	return RemoveConfigDirective{
		KV: sshconfig.KV{
			Key:   key,
			Value: filepath.ToSlash(value),
		},
	}
}

func (d RemoveConfigDirective) Apply(host *sshconfig.Host) {
	for i, node := range host.Nodes {
		if directiveMatches(node, d.KV) {
			host.Nodes = append(host.Nodes[:i], host.Nodes[i+1:]...)
			return
		}
	}
}

func readConfigFile(path string) (*sshconfig.Config, error) {
	cfgFile, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to open topo ssh config file: %w", err)
	}
	defer func() {
		if cfgFile != nil {
			if err := cfgFile.Close(); err != nil {
				logger.Error("failed to close topo ssh config file", "error", err)
			}
		}
	}()

	cfgReader := io.Reader(cfgFile)
	if cfgFile == nil {
		cfgReader = strings.NewReader("")
	}

	cfg, err := sshconfig.Decode(cfgReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode topo ssh config file: %w", err)
	}

	return cfg, nil
}

func findOrCreateHostBlock(cfg *sshconfig.Config, alias string) (*sshconfig.Host, error) {
	// all configs start with a default 'Host *' declaration
	if alias == "" {
		return cfg.Hosts[0], nil
	}

	for _, host := range cfg.Hosts[1:] {
		if host.Matches(alias) {
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

func directiveMatches(node sshconfig.Node, directive sshconfig.KV) bool {
	if kv, ok := node.(*sshconfig.KV); ok {
		return kv.Key == directive.Key
	}

	if include, ok := node.(*sshconfig.Include); ok {
		return include.String() == directive.String()
	}

	return false
}

func updateConfigFile(path string, host string, modifiers []ConfigDirectiveModifier) error {
	cfg, err := readConfigFile(path)
	if err != nil {
		return err
	}

	hostBlock, err := findOrCreateHostBlock(cfg, host)
	if err != nil {
		return fmt.Errorf("failed to find or create host block: %w", err)
	}

	for _, modifier := range modifiers {
		modifier.Apply(hostBlock)
	}

	cfgBytes, err := cfg.MarshalText()
	if err != nil {
		return fmt.Errorf("failed to marshal ssh config: %w", err)
	}

	if err := os.WriteFile(path, cfgBytes, 0o600); err != nil {
		return fmt.Errorf("failed to write to %s: %w", path, err)
	}
	return nil
}

func GetConfigDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory for SSH config: %w", err)
	}

	return filepath.Join(home, ".ssh"), nil
}

func CreateOrModifyConfigFile(sshDir string, dest Destination, modifiers []ConfigDirectiveModifier) error {
	topoConfigPath := filepath.Join(sshDir, TopoConfigFileName)
	if err := updateConfigFile(topoConfigPath, dest.Host, modifiers); err != nil {
		return err
	}

	defaultConfigPath := filepath.Join(sshDir, DefaultConfigFileName)
	return updateConfigFile(defaultConfigPath, "", []ConfigDirectiveModifier{
		NewEnsureConfigDirectivePath("Include", topoConfigPath),
	})
}

func LegacyTopoConfigDirectoryExists(sshDir string) (bool, error) {
	info, err := os.Stat(filepath.Join(sshDir, TopoConfigFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check for legacy topo ssh config file: %w", err)
	}

	if info.IsDir() {
		return true, nil
	}

	return false, nil
}

func MigrateLegacyTopoConfig(sshDir string) error {
	legacyDir := filepath.Join(sshDir, TopoConfigFileName)
	if exists, err := LegacyTopoConfigDirectoryExists(sshDir); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("legacy topo ssh config directory not found at %s; nothing to migrate", legacyDir)
	}

	legacyGlob := filepath.Join(legacyDir, "*.conf")
	confFiles, err := filepath.Glob(legacyGlob)
	if err != nil {
		return fmt.Errorf("failed to list config files in %s: %w", legacyDir, err)
	}

	var combined []byte
	for _, confFile := range confFiles {
		content, err := os.ReadFile(confFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", confFile, err)
		}
		combined = append(combined, content...)
	}

	unifiedPath := legacyDir + ".new"
	// #nosec G703 -- ssh config is always stored in the user's home directory
	if err := os.WriteFile(unifiedPath, combined, 0o600); err != nil {
		return fmt.Errorf("failed to write unified config to %s: %w", unifiedPath, err)
	}

	if err := os.RemoveAll(legacyDir); err != nil {
		return fmt.Errorf("failed to remove legacy config directory %s: %w", legacyDir, err)
	}

	if err := os.Rename(unifiedPath, legacyDir); err != nil {
		return fmt.Errorf("failed to move migrated config to %s: %w", legacyDir, err)
	}

	defaultConfigPath := filepath.Join(sshDir, DefaultConfigFileName)
	return updateConfigFile(defaultConfigPath, "", []ConfigDirectiveModifier{
		NewRemoveConfigDirectivePath("Include", legacyGlob),
		NewEnsureConfigDirectivePath("Include", legacyDir),
	})
}
