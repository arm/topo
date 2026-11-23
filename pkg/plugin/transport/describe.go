package transport

import (
	"context"
	"fmt"
	"net/rpc"
	"strings"

	goplugin "github.com/hashicorp/go-plugin"
)

const DescribeKey = "describe"

type Descriptor struct {
	ID         string
	Capability Capability
}

func (d Descriptor) Validate() error {
	publisher, name, ok := strings.Cut(d.ID, ".")
	if !ok || publisher == "" || name == "" || strings.Contains(name, ".") {
		return fmt.Errorf("plugin identity must be <publisher>.<name>: %q", d.ID)
	}
	if publisher == "builtin" {
		return fmt.Errorf("plugin publisher %q is reserved", publisher)
	}
	return nil
}

// Describer describes the executable's identity and provider capability without probing.
type Describer interface {
	Describe(context.Context) (Descriptor, error)
}

// DescribeRPCPlugin exposes only plugin identity and provider capability. Impl is set only by the server.
type DescribeRPCPlugin struct {
	Impl Describer
}

func (p *DescribeRPCPlugin) Server(_ *goplugin.MuxBroker) (any, error) {
	return &describeServer{impl: p.Impl}, nil
}

func (p *DescribeRPCPlugin) Client(_ *goplugin.MuxBroker, client *rpc.Client) (any, error) {
	return &describeClient{client: client}, nil
}

type describeClient struct {
	client *rpc.Client
}

func (p *describeClient) Describe(ctx context.Context) (Descriptor, error) {
	var result Descriptor
	err := call(ctx, p.client, "Plugin.Describe", struct{}{}, &result)
	return result, err
}

type describeServer struct {
	impl Describer
}

func (p *describeServer) Describe(_ struct{}, reply *Descriptor) error {
	result, err := p.impl.Describe(context.Background())
	*reply = result
	return err
}
