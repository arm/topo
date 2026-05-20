package deploy

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/arm/topo/internal/deploy/command"
)

type Container struct {
	Image   string `json:"Image"`
	Status  string `json:"Status"`
	Address string `json:"Address"`
}

type rawContainer struct {
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

func ListRunningContainers(composeFile string, h command.Host, hostName string) ([]Container, error) {
	var buf bytes.Buffer
	cmd := command.DockerCompose(h, composeFile, "ps", "--format", "json")
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	containers := []Container{}
	decoder := json.NewDecoder(&buf)
	for decoder.More() {
		var raw rawContainer
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		address := raw.Ports
		if hostName != "" {
			address = publishedAddress(raw.Ports, hostName)
		}
		containers = append(containers, Container{
			Image:   raw.Image,
			Status:  raw.Status,
			Address: address,
		})
	}
	return containers, nil
}

func publishedAddress(rawPorts, hostName string) string {
	address := rawPorts
	if i := strings.Index(address, "->"); i != -1 {
		address = address[:i]
	}
	return strings.ReplaceAll(address, "0.0.0.0", hostName)
}
