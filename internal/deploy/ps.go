package deploy

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/arm/topo/internal/deploy/command"
)

type Container struct {
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

func ListRunningContainers(composeFile string, h command.Host) ([]Container, error) {
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
		var c Container
		if err := decoder.Decode(&c); err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}
	return containers, nil
}

func PublishedAddress(rawPorts, hostName string) string {
	address := rawPorts
	if i := strings.Index(address, "->"); i != -1 {
		address = address[:i]
	}
	return strings.ReplaceAll(address, "0.0.0.0", hostName)
}
