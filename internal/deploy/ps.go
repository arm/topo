package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
)

func PS(
	ctx context.Context,
	scope project.Scope,
	target ssh.Destination,
	engine EngineKind,
	hostname string,
	all bool,
) (containers []EnrichedContainer, err error) {
	if engine != EngineDocker && engine != EnginePodman {
		return nil, fmt.Errorf("invalid engine %q: must be docker or podman", engine)
	}

	var runner ComposeCommandRunner
	host := docker.NewHostFromDestination(target)
	if engine == EngineDocker {
		runner = docker.Engine{Host: host}
	} else {
		socket := podman.LocalSocket
		if !target.IsPlainLocalhost() {
			tunnel, err := podman.TunnelRemoteSocketPath(ctx, io.Discard, target)
			if err != nil {
				return nil, err
			}
			defer func() {
				err = errors.Join(err, tunnel.Close())
			}()
			socket = podman.NewSocket(tunnel.SocketURL())
		}
		runner = podman.Engine{Socket: socket}
	}

	raws, err := ListContainers(ctx, scope, runner, all)
	if err != nil {
		return nil, err
	}
	var domains map[string]string
	if engine == EngineDocker {
		domains, err = docker.ProcessingDomains(ctx, host, containerIDs(raws))
		if err != nil {
			return nil, err
		}
	}
	return enrichContainers(raws, hostname, domains), nil
}

func containerIDs(raws []RawContainer) []string {
	ids := make([]string, 0, len(raws))
	for _, raw := range raws {
		if raw.ID != "" {
			ids = append(ids, raw.ID)
		}
	}
	return ids
}

const hostProcessingDomain = "Linux Host"

type EnrichedContainer struct {
	Id               string
	Names            string
	Image            string
	State            string
	Status           string
	ProcessingDomain string
	Address          string
}

func enrichContainers(raws []RawContainer, hostname string, domains map[string]string) []EnrichedContainer {
	containers := make([]EnrichedContainer, len(raws))
	for i, raw := range raws {
		domain, ok := domains[raw.ID]
		if !ok {
			domain = hostProcessingDomain
		}
		containers[i] = EnrichedContainer{
			Id:               raw.ID,
			Names:            raw.Names,
			Image:            raw.Image,
			State:            raw.State,
			Status:           raw.Status,
			ProcessingDomain: domain,
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
