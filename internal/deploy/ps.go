package deploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/deploy/command"
)

const (
	remoteprocRuntimeName    = "io.containerd.remoteproc.v1"
	remoteprocNameAnnotation = "remoteproc.name"
	hostProcessingDomain     = "Linux Host"
	dockerShortIDLength      = 12
)

type PSContainer struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

type Container struct {
	Id               string `json:"id"`
	Names            string `json:"names"`
	Image            string `json:"image"`
	Status           string `json:"status"`
	ProcessingDomain string `json:"processingDomain"`
	Address          string `json:"address"`
}

type InspectedContainer struct {
	ID         string              `json:"Id"`
	Name       string              `json:"Name"`
	HostConfig InspectedHostConfig `json:"HostConfig"`
}

type InspectedHostConfig struct {
	Runtime     string            `json:"Runtime"`
	Annotations map[string]string `json:"Annotations"`
}

func ListContainers(composeFile string, h command.Host, hostName string, all bool) ([]Container, error) {
	rawJSON, err := getContainers(composeFile, h, all)
	if err != nil {
		return nil, err
	}
	raws, err := ParseContainers(rawJSON)
	if err != nil {
		return nil, err
	}
	containers := RemapAddresses(raws, hostName)

	domains, err := getProcessingDomains(raws, h)
	if err != nil {
		return nil, err
	}

	for i, raw := range raws {
		containers[i].ProcessingDomain = processingDomainFromLookup(raw, domains)
	}
	return containers, nil
}

func getContainers(composeFile string, h command.Host, all bool) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := command.DockerCompose(h, composeFile, composePSArgs(all)...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker compose ps: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func composePSArgs(all bool) []string {
	args := []string{"ps", "--format", "json"}
	if all {
		args = append(args, "--all")
	}
	return args
}

func ParseContainers(rawJSON string) ([]PSContainer, error) {
	raws := []PSContainer{}
	decoder := json.NewDecoder(strings.NewReader(rawJSON))
	for decoder.More() {
		var raw PSContainer
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		raws = append(raws, raw)
	}
	return raws, nil
}

func getProcessingDomains(raws []PSContainer, h command.Host) (map[string]string, error) {
	targets := make([]string, 0, len(raws))
	for _, raw := range raws {
		if raw.ID != "" {
			targets = append(targets, raw.ID)
		}
	}
	if len(targets) == 0 {
		return map[string]string{}, nil
	}

	rawJSON, err := inspectContainers(targets, h)
	if err != nil {
		return nil, err
	}

	inspected, err := ParseInspectedContainers(rawJSON)
	if err != nil {
		return nil, err
	}

	return BuildProcessingDomainLookup(inspected), nil
}

func inspectContainers(targets []string, h command.Host) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := command.Docker(h, append([]string{"inspect"}, targets...)...)
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

func RemapAddresses(raws []PSContainer, hostName string) []Container {
	containers := make([]Container, len(raws))
	for i, raw := range raws {
		containers[i] = Container{
			Id:      raw.ID,
			Names:   raw.Names,
			Image:   raw.Image,
			Status:  raw.Status,
			Address: publishedAddress(raw.Ports, hostName),
		}
	}
	return containers
}

func processingDomainFromLookup(raw PSContainer, lookup map[string]string) string {
	if domain, ok := lookup[raw.ID]; ok {
		return domain
	}
	return hostProcessingDomain
}

func publishedAddress(rawPorts, hostName string) string {
	if hostName == "" {
		return rawPorts
	}
	parts := strings.Split(rawPorts, ", ")
	for i, part := range parts {
		if idx := strings.Index(part, "->"); idx != -1 {
			part = part[:idx]
		}
		parts[i] = strings.ReplaceAll(part, "0.0.0.0", hostName)
	}
	return strings.Join(parts, ", ")
}
