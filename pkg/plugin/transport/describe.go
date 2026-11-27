package transport

import (
	"context"
	"fmt"
	"strings"

	"github.com/arm/topo/pkg/plugin/transport/pluginpb"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
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

type DescribeRPCPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	pluginpb.UnimplementedDescribeServiceServer
	Descriptor Descriptor
}

func (p *DescribeRPCPlugin) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	pluginpb.RegisterDescribeServiceServer(server, p)
	return nil
}

func (p *DescribeRPCPlugin) GRPCClient(
	_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn,
) (any, error) {
	return &DescribeClient{rpc: pluginpb.NewDescribeServiceClient(conn)}, nil
}

func (p *DescribeRPCPlugin) Describe(context.Context, *emptypb.Empty) (*pluginpb.Descriptor, error) {
	return &pluginpb.Descriptor{Id: p.Descriptor.ID, Capability: string(p.Descriptor.Capability)}, nil
}

type DescribeClient struct {
	rpc pluginpb.DescribeServiceClient
}

func (p *DescribeClient) Describe(ctx context.Context) (Descriptor, error) {
	reply, err := p.rpc.Describe(ctx, &emptypb.Empty{})
	if err != nil {
		return Descriptor{}, callError(ctx, err)
	}
	return Descriptor{ID: reply.GetId(), Capability: Capability(reply.GetCapability())}, nil
}
