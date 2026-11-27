package plugins

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
	goplugin "github.com/hashicorp/go-plugin"
)

// Process is a running suggestion plugin and its RPC connection.
type Process struct {
	descriptor  transport.Descriptor
	client      *goplugin.Client
	suggestions suggestions.Provider
}

// Load starts a suggestion plugin executable and validates its descriptor.
// Cancelling ctx stops the process. Call Close when the process is no longer needed.
func Load(ctx context.Context, path string) (_ *Process, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: transport.Handshake,
		Plugins: goplugin.PluginSet{
			transport.DescribeKey:         &transport.DescribeRPCPlugin{},
			string(transport.Suggestions): &transport.SuggestionsRPCPlugin{},
		},
		Cmd:              exec.CommandContext(ctx, path),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		StartTimeout:     10 * time.Second,
		AutoMTLS:         true,
	})
	defer func() {
		if err != nil {
			client.Kill()
		}
	}()

	connection, err := client.Client()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("starting plugin %q: %w", path, err)
	}

	raw, err := connection.Dispense(transport.DescribeKey)
	if err != nil {
		return nil, fmt.Errorf("obtaining describe adapter from %q: %w", path, err)
	}
	descriptor, err := raw.(*transport.DescribeClient).Describe(ctx)
	if err != nil {
		return nil, fmt.Errorf("describing plugin %q: %w", path, err)
	}
	if err := descriptor.Validate(); err != nil {
		return nil, fmt.Errorf("invalid descriptor from plugin %q: %w", path, err)
	}
	if descriptor.Capability != transport.Suggestions {
		return nil, fmt.Errorf("plugin %q advertised unsupported capability %q", path, descriptor.Capability)
	}

	raw, err = connection.Dispense(string(descriptor.Capability))
	if err != nil {
		return nil, fmt.Errorf("plugin %q advertised %s but its adapter failed: %w", path, descriptor.Capability, err)
	}
	return &Process{descriptor: descriptor, client: client, suggestions: raw.(suggestions.Provider)}, nil
}

// Descriptor returns the validated identity and capability reported at startup.
func (p *Process) Descriptor() transport.Descriptor {
	return p.descriptor
}

// Close stops the plugin and waits for it to exit.
func (p *Process) Close() {
	p.client.Kill()
}

// Exited reports whether the plugin process has exited.
func (p *Process) Exited() bool {
	return p.client.Exited()
}

func (p *Process) Suggestions(ctx context.Context, request suggestions.Request) (suggestions.Response, error) {
	result, err := p.suggestions.Suggestions(ctx, request)
	if err != nil {
		return suggestions.Response{}, fmt.Errorf("listing suggestions from plugin %q: %w", p.descriptor.ID, err)
	}
	return result, nil
}

func (p *Process) RunAction(
	ctx context.Context, id suggestions.ActionID, request suggestions.Request,
) (suggestions.ActionResponse, error) {
	result, err := p.suggestions.RunAction(ctx, id, request)
	if err != nil {
		return suggestions.ActionResponse{}, fmt.Errorf("running action %q on plugin %q: %w", id, p.descriptor.ID, err)
	}
	return result, nil
}
