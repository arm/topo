// Package transport exposes the low-level RPC protocol used by the SDK and Topo's plugin host.
// Plugin authors should use plugin.ServeSuggestions instead.
package transport

import (
	"context"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc/status"
)

// Capability identifies the kind of provider exposed by an executable.
type Capability string

const Suggestions Capability = "suggestions"

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "TOPO_PLUGIN",
	MagicCookieValue: "topo-plugin-v1",
}

func callError(ctx context.Context, err error) error {
	if local := ctx.Err(); local != nil && status.Code(err) == status.FromContextError(local).Code() {
		return local
	}
	return err
}
