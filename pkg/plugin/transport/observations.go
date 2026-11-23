package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/rpc"

	"github.com/arm/topo/pkg/plugin/observations"
	goplugin "github.com/hashicorp/go-plugin"
)

// ObservationsRPCPlugin exposes the optional observations contract independently of the base adapter.
// Impl is set only by the server.
type ObservationsRPCPlugin struct {
	Impl observations.Provider
}

func (p *ObservationsRPCPlugin) Server(broker *goplugin.MuxBroker) (any, error) {
	return &observationsServer{impl: p.Impl, broker: broker}, nil
}

func (p *ObservationsRPCPlugin) Client(broker *goplugin.MuxBroker, client *rpc.Client) (any, error) {
	return &observationsClient{client: client, broker: broker}, nil
}

type observationsClient struct {
	client *rpc.Client
	broker *goplugin.MuxBroker
}

func (p *observationsClient) ObserveTarget(
	ctx context.Context, runner observations.TargetRunner,
) (observations.Tree, error) {
	var result []byte
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	callbackID := p.broker.NextId()
	go p.broker.AcceptAndServe(callbackID, &runnerServer{ctx: ctx, impl: runner})
	if err := call(ctx, p.client, "Plugin.ObserveTarget", callbackID, &result); err != nil {
		return nil, err
	}
	return decodeObservations(result)
}

type observationsServer struct {
	impl   observations.Provider
	broker *goplugin.MuxBroker
}

func (p *observationsServer) ObserveTarget(callbackID uint32, reply *[]byte) (err error) {
	connection, err := p.broker.Dial(callbackID)
	if err != nil {
		return err
	}
	client := rpc.NewClient(connection)
	defer func() { err = errors.Join(err, client.Close()) }()
	result, err := p.impl.ObserveTarget(context.Background(), &runnerClient{client: client})
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("invalid observations: %w", err)
	}
	*reply = encoded
	return nil
}

func decodeObservations(data []byte) (observations.Tree, error) {
	var result observations.Tree
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding observations: %w", err)
	}
	return result, nil
}

// RunReply carries execution errors as data so stdout and stderr are preserved.
type RunReply struct {
	Stdout string
	Stderr string
	Error  string
}

type runnerClient struct {
	client *rpc.Client
}

func (r *runnerClient) Run(ctx context.Context, command string) (string, string, error) {
	var result RunReply
	if err := call(ctx, r.client, "Plugin.Run", command, &result); err != nil {
		return "", "", err
	}
	if result.Error != "" {
		return result.Stdout, result.Stderr, fmt.Errorf("%s", result.Error)
	}
	return result.Stdout, result.Stderr, nil
}

type runnerServer struct {
	ctx  context.Context
	impl observations.TargetRunner
}

func (r *runnerServer) Run(command string, reply *RunReply) error {
	stdout, stderr, err := r.impl.Run(r.ctx, command)
	*reply = RunReply{Stdout: stdout, Stderr: stderr}
	if err != nil {
		reply.Error = err.Error()
	}
	return nil
}
