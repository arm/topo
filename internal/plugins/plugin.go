package plugins

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
	goplugin "github.com/hashicorp/go-plugin"
)

// process owns one local plugin process and its metadata.
type process struct {
	path       string
	descriptor transport.Descriptor
	connection goplugin.ClientProtocol
	client     *goplugin.Client
}

// load starts one executable and validates its descriptor without probing.
func load(ctx context.Context, path string) (result *process, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: transport.Handshake,
		Plugins: goplugin.PluginSet{
			transport.DescribeKey:          &transport.DescribeRPCPlugin{},
			string(transport.Observations): &transport.ObservationsRPCPlugin{},
			string(transport.Suggestions):  &transport.SuggestionsRPCPlugin{},
		},
		Cmd:              exec.CommandContext(ctx, path),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolNetRPC},
		StartTimeout:     10 * time.Second,
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
	describer, ok := raw.(transport.Describer)
	if !ok {
		return nil, fmt.Errorf("plugin %q has an invalid describe RPC adapter", path)
	}
	descriptor, err := describer.Describe(ctx)
	if err != nil {
		return nil, fmt.Errorf("describing plugin %q: %w", path, err)
	}
	if err := descriptor.Validate(); err != nil {
		return nil, fmt.Errorf("invalid descriptor from plugin %q: %w", path, err)
	}
	return &process{path: path, descriptor: descriptor, connection: connection, client: client}, nil
}

func (p *process) Close() {
	p.client.Kill()
}

func (p *process) ObserveTarget(ctx context.Context, runner observations.TargetRunner) (observations.Tree, error) {
	if p.descriptor.Capability != transport.Observations {
		return observations.Tree{}, fmt.Errorf("plugin %q does not advertise %s", p.path, transport.Observations)
	}
	raw, err := p.connection.Dispense(string(transport.Observations))
	if err != nil {
		return observations.Tree{}, fmt.Errorf("plugin %q advertised %s but its observations adapter failed: %w",
			p.path, transport.Observations, err)
	}
	provider, ok := raw.(observations.Provider)
	if !ok {
		return observations.Tree{}, fmt.Errorf("plugin %q has an invalid observations RPC adapter", p.path)
	}
	result, err := provider.ObserveTarget(ctx, runner)
	if err != nil {
		return observations.Tree{}, fmt.Errorf("observations plugin %q: %w", p.path, err)
	}
	return result, nil
}

// Suggestions returns candidates and actions without running any action.
func (p *process) Suggestions(
	ctx context.Context, request suggestions.Request,
) (suggestions.Response, error) {
	provider, err := p.suggestionsProvider()
	if err != nil {
		return suggestions.Response{}, err
	}
	result, err := provider.Suggestions(ctx, request)
	if err != nil {
		return suggestions.Response{}, fmt.Errorf("suggestions plugin %q: %w", p.path, err)
	}
	return result, nil
}

// RunAction is called only after a user selects an action, with a fresh request.
func (p *process) RunAction(
	ctx context.Context, id suggestions.ActionID, request suggestions.Request,
) (suggestions.ActionResponse, error) {
	provider, err := p.suggestionsProvider()
	if err != nil {
		return suggestions.ActionResponse{}, err
	}
	result, err := provider.RunAction(ctx, id, request)
	if err != nil {
		return suggestions.ActionResponse{}, fmt.Errorf("plugin %q action %q: %w", p.path, id, err)
	}
	return result, nil
}

func (p *process) suggestionsProvider() (suggestions.Provider, error) {
	if p.descriptor.Capability != transport.Suggestions {
		return nil, fmt.Errorf("plugin %q does not advertise %s", p.path, transport.Suggestions)
	}
	raw, err := p.connection.Dispense(string(transport.Suggestions))
	if err != nil {
		return nil, fmt.Errorf("plugin %q advertised %s but its suggestions adapter failed: %w",
			p.path, transport.Suggestions, err)
	}
	provider, ok := raw.(suggestions.Provider)
	if !ok {
		return nil, fmt.Errorf("plugin %q has an invalid suggestions RPC adapter", p.path)
	}
	return provider, nil
}
