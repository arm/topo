package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	remoteprocRuntimeName    = "io.containerd.remoteproc.v1"
	remoteprocNameAnnotation = "remoteproc.name"
	dockerShortIDLength      = 12
)

type InspectedContainer struct {
	ID         string              `json:"Id"`
	Name       string              `json:"Name"`
	HostConfig InspectedHostConfig `json:"HostConfig"`
}

type InspectedHostConfig struct {
	Runtime     string            `json:"Runtime"`
	Annotations map[string]string `json:"Annotations"`
}

func ProcessingDomains(ctx context.Context, h Host, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	rawJSON, err := inspectContainers(ctx, ids, h)
	if err != nil {
		return nil, err
	}
	inspected, err := ParseInspectedContainers(rawJSON)
	if err != nil {
		return nil, err
	}
	return BuildProcessingDomainLookup(inspected), nil
}

func inspectContainers(ctx context.Context, targets []string, h Host) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := Command(ctx, h, append([]string{"inspect"}, targets...)...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker inspect: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func ParseInspectedContainers(rawJSON string) ([]InspectedContainer, error) {
	if strings.TrimSpace(rawJSON) == "" {
		return []InspectedContainer{}, nil
	}

	inspected := []InspectedContainer{}
	if err := json.Unmarshal([]byte(rawJSON), &inspected); err != nil {
		return nil, err
	}
	return inspected, nil
}

func BuildProcessingDomainLookup(inspected []InspectedContainer) map[string]string {
	lookup := map[string]string{}

	for _, container := range inspected {
		if container.HostConfig.Runtime != remoteprocRuntimeName {
			continue
		}

		processingDomain := strings.TrimSpace(container.HostConfig.Annotations[remoteprocNameAnnotation])
		if processingDomain == "" {
			continue
		}

		if id := strings.TrimSpace(container.ID); id != "" {
			if len(id) > dockerShortIDLength {
				id = id[:dockerShortIDLength]
			}
			lookup[id] = processingDomain
		}
	}

	return lookup
}
