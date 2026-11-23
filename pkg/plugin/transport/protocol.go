// Package transport exposes the low-level RPC protocol used by the SDK and Topo's plugin host.
// Plugin authors should use plugin.ServeObservations or plugin.ServeSuggestions instead.
package transport

import goplugin "github.com/hashicorp/go-plugin"

// Capability identifies the kind of provider exposed by an executable.
type Capability string

const (
	Observations Capability = "observations"
	Suggestions  Capability = "suggestions"
)

// The magic cookie identifies the plugin family; it is not authentication.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "TOPO_PLUGIN",
	MagicCookieValue: "topo-plugin-v1",
}
