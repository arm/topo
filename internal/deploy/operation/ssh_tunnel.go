package operation

import (
	"context"
	"fmt"
	"io"

	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
)

type registrySSHTunnelState struct {
	tunnel *ssh.SSHTunnel
}

type registrySSHTunnelStart struct {
	targetDest        ssh.Destination
	port              string
	useControlSockets bool
	state             *registrySSHTunnelState
}

type registrySSHTunnelStop struct {
	state *registrySSHTunnelState
}

func NewRegistrySSHTunnel(targetDest ssh.Destination, port string, useControlSockets bool) (goperation.Operation, goperation.Operation) {
	state := new(registrySSHTunnelState)
	start := &registrySSHTunnelStart{
		targetDest:        targetDest,
		port:              port,
		useControlSockets: useControlSockets,
		state:             state,
	}
	return start, &registrySSHTunnelStop{state: state}
}

func (s *registrySSHTunnelStart) Description() string {
	return "Open registry SSH tunnel"
}

func (s *registrySSHTunnelStart) Run(w io.Writer) error {
	tunnel, err := ssh.OpenSSHTunnel(context.Background(), w, s.targetDest, s.port, s.useControlSockets)
	if err != nil {
		return fmt.Errorf("failed to open SSH tunnel: %w; ensure port %s is free or specify a different one with `--registry-port`", err, s.port)
	}
	s.state.tunnel = tunnel
	return nil
}

func (s *registrySSHTunnelStop) Description() string {
	return "Close registry SSH tunnel"
}

func (s *registrySSHTunnelStop) Run(w io.Writer) error {
	if s.state.tunnel == nil {
		return nil
	}
	if err := s.state.tunnel.Close(context.Background(), w); err != nil {
		return fmt.Errorf("failed to close SSH tunnel: %w", err)
	}
	s.state.tunnel = nil
	return nil
}
