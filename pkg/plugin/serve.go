// Package plugin provides registration and serving for Go-only Topo plugins.
package plugin

import (
	"context"

	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
	goplugin "github.com/hashicorp/go-plugin"
)

func ServeObservations(identity string, provider observations.Provider) {
	serve(identity, transport.Observations, &transport.ObservationsRPCPlugin{Impl: provider})
}

func ServeSuggestions(identity string, provider suggestions.Provider) {
	serve(identity, transport.Suggestions, &transport.SuggestionsRPCPlugin{Impl: provider})
}

func serve(identity string, capability transport.Capability, adapter goplugin.Plugin) {
	descriptor := transport.Descriptor{ID: identity, Capability: capability}
	if err := descriptor.Validate(); err != nil {
		panic(err)
	}
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: transport.Handshake,
		Plugins: goplugin.PluginSet{
			transport.DescribeKey: &transport.DescribeRPCPlugin{Impl: registration{descriptor: descriptor}},
			string(capability):    adapter,
		},
	})
}

type registration struct {
	descriptor transport.Descriptor
}

func (r registration) Describe(context.Context) (transport.Descriptor, error) {
	return r.descriptor, nil
}
