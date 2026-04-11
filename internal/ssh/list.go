package ssh

import (
	"strings"

	"github.com/arm/topo/internal/collections"
	"github.com/arm/topo/internal/output/logger"
	sshconfig "github.com/kevinburke/ssh_config"
)

func gatherIncludedConfigPaths(cfg *sshconfig.Config) []string {
	includedPaths := []string{}

	for _, host := range cfg.Hosts {
		for _, node := range host.Nodes {
			if include, ok := node.(*sshconfig.Include); ok {
				includePath := strings.TrimSpace(strings.TrimPrefix(include.String(), "Include"))
				if includePath != "" {
					includedPaths = append(includedPaths, includePath)
				}
			}
		}
	}

	return includedPaths
}

func ListHosts(configPath string) []string {
	queue := []string{configPath}
	seen := collections.NewSet[string]()
	hosts := collections.NewSet[string]()

	for len(queue) > 0 {
		currentPath := queue[0]
		queue = queue[1:]
		if seen.Contains(currentPath) {
			continue
		}
		seen.Add(currentPath)

		cfg, err := readConfigFile(currentPath)
		if err != nil {
			logger.Error("failed to read ssh config file while listing hosts", "path", currentPath, "error", err)
			continue
		}

		for _, host := range cfg.Hosts {
			queue = append(queue, gatherIncludedConfigPaths(cfg)...)

			for _, pattern := range host.Patterns {
				patternStr := pattern.String()
				if patternStr == "*" {
					continue
				}
				hosts.Add(patternStr)
			}
		}
	}

	return hosts.ToSlice()
}
