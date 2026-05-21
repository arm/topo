package deploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/deploy/command"
)

type RawContainer struct {
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

type Container struct {
	Image   string `json:"Image"`
	Status  string `json:"Status"`
	Address string `json:"Address"`
}

func ListRunningContainers(composeFile string, h command.Host, hostName string) ([]Container, error) {
	rawJSON, err := getRunningContainers(composeFile, h)
	if err != nil {
		return nil, err
	}
	raws, err := ParseRunningContainers(rawJSON)
	if err != nil {
		return nil, err
	}
	return RemapAddresses(raws, hostName), nil
}

func getRunningContainers(composeFile string, h command.Host) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := command.DockerCompose(h, composeFile, "ps", "--format", "json")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker compose ps: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func ParseRunningContainers(rawJSON string) ([]RawContainer, error) {
	raws := []RawContainer{}
	decoder := json.NewDecoder(strings.NewReader(rawJSON))
	for decoder.More() {
		var raw RawContainer
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		raws = append(raws, raw)
	}
	return raws, nil
}

func RemapAddresses(raws []RawContainer, hostName string) []Container {
	containers := make([]Container, len(raws))
	for i, raw := range raws {
		containers[i] = Container{
			Image:   raw.Image,
			Status:  raw.Status,
			Address: publishedAddress(raw.Ports, hostName),
		}
	}
	return containers
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
