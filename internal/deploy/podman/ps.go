package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/ssh"
)

const hostProcessingDomain = "Linux Host"

type PSContainer struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

type Container struct {
	Id               string `json:"id"`
	Names            string `json:"names"`
	Image            string `json:"image"`
	State            string `json:"state"`
	Status           string `json:"status"`
	ProcessingDomain string `json:"processingDomain"`
	Address          string `json:"address"`
}

func ListContainers(composeFile string, target ssh.Destination, hostname string, all bool) (containers []Container, err error) {
	socket := LocalSocket
	var tunnel *ssh.TCPToUnixSocketTunnel
	if !target.IsPlainLocalhost() {
		tunnel, err = TunnelRemoteSocketPath(context.Background(), io.Discard, target)
		if err != nil {
			return nil, err
		}
		defer func() {
			err = errors.Join(err, tunnel.Close())
		}()
		socket = NewSocket(tunnel.SocketURL())
	}

	rawJSON, err := getContainers(composeFile, socket, all)
	if err != nil {
		return nil, err
	}
	raws, err := ParseContainers(rawJSON)
	if err != nil {
		return nil, err
	}
	return RemapAddresses(raws, hostname), nil
}

func getContainers(composeFile string, socket Socket, all bool) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd, err := ComposeCommand(context.Background(), socket, composeFile, composePSArgs(all)...)
	if err != nil {
		return "", err
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("podman compose ps: %w: %s", err, strings.TrimSpace(stderr.String()))
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
	for {
		var raw PSContainer
		err := decoder.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		raws = append(raws, raw)
	}
	return raws, nil
}

func RemapAddresses(raws []PSContainer, hostname string) []Container {
	containers := make([]Container, len(raws))
	for i, raw := range raws {
		containers[i] = Container{
			Id:               raw.ID,
			Names:            raw.Names,
			Image:            raw.Image,
			State:            raw.State,
			Status:           raw.Status,
			ProcessingDomain: hostProcessingDomain,
			Address:          publishedAddress(raw.Ports, hostname),
		}
	}
	return containers
}

func publishedAddress(rawPorts, hostname string) string {
	if hostname == "" {
		return rawPorts
	}
	parts := strings.Split(rawPorts, ", ")
	for i, part := range parts {
		if idx := strings.Index(part, "->"); idx != -1 {
			part = part[:idx]
		}
		parts[i] = strings.ReplaceAll(part, "0.0.0.0", hostname)
	}
	return strings.Join(parts, ", ")
}
