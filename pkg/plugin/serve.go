// Package plugin provides registration and serving for Go-only Topo plugins.
package plugin

import (
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
	goplugin "github.com/hashicorp/go-plugin"
)

// ServeSuggestions serves a suggestion provider to the Topo process that launched it.
func ServeSuggestions(identity string, provider suggestions.Provider) {
	descriptor := transport.Descriptor{ID: identity, Capability: transport.Suggestions}
	if err := descriptor.Validate(); err != nil {
		panic(err)
	}
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: transport.Handshake,
		GRPCServer:      goplugin.DefaultGRPCServer,
		Plugins: goplugin.PluginSet{
			transport.DescribeKey:         &transport.DescribeRPCPlugin{Descriptor: descriptor},
			string(transport.Suggestions): &transport.SuggestionsRPCPlugin{Impl: provider},
		},
	})
}
