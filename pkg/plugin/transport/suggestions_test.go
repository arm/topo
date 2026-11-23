package transport_test

import (
	"context"
	"net"
	"net/rpc"
	"strconv"
	"testing"
	"time"

	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuggestionsRPCPlugin(t *testing.T) {
	for _, method := range []string{"Suggestions", "RunAction"} {
		t.Run(method, func(t *testing.T) {
			requests := make(chan suggestions.Request, 1)
			provider := newSuggestionsRPCClient(t, observationConsumer{requests: requests})
			request := suggestions.Request{
				Project: suggestions.ProjectMetadata{Name: "Image segmentation"},
				TargetObservations: observations.Tree{
					"arm": observations.Tree{"fancy": observations.Tree{
						"target": observations.Tree{"cpu_count": int64(9007199254740993)},
					}},
				}}

			var got []suggestions.Suggestion
			var err error
			if method == "Suggestions" {
				response, callErr := provider.Suggestions(context.Background(), request)
				got, err = response.Suggestions, callErr
			} else {
				response, callErr := provider.RunAction(context.Background(), "use-count", request)
				got, err = response.Suggestions, callErr
			}

			require.NoError(t, err)
			assert.Equal(t, []suggestions.Suggestion{{Value: "9007199254740993"}}, got)
			assert.Equal(t, suggestions.ProjectMetadata{Name: "Image segmentation"}, (<-requests).Project)
		})
	}
}

func newSuggestionsRPCClient(t *testing.T, impl suggestions.Provider) suggestions.Provider {
	t.Helper()
	adapter := &transport.SuggestionsRPCPlugin{Impl: impl}
	server, err := adapter.Server(nil)
	require.NoError(t, err)
	rpcServer := rpc.NewServer()
	require.NoError(t, rpcServer.RegisterName("Plugin", server))
	clientConnection, serverConnection := net.Pipe()
	t.Cleanup(func() { _ = serverConnection.Close() })
	go rpcServer.ServeConn(serverConnection)
	rpcClient := rpc.NewClient(clientConnection)
	t.Cleanup(func() { _ = rpcClient.Close() })
	client, err := adapter.Client(nil, rpcClient)
	require.NoError(t, err)
	return client.(suggestions.Provider)
}

func TestSuggestionsActionCancellation(t *testing.T) {
	t.Run("stops plugin work and keeps the connection usable", func(t *testing.T) {
		impl := &cancellableSuggestions{started: make(chan struct{}), stopped: make(chan struct{})}
		provider := newSuggestionsRPCClient(t, impl)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		result := make(chan error, 1)
		go func() {
			_, err := provider.RunAction(ctx, "wait", suggestions.Request{})
			result <- err
		}()
		select {
		case <-impl.started:
		case <-time.After(time.Second):
			t.Fatal("action did not start")
		}

		cancel()

		select {
		case err := <-result:
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(3 * time.Second):
			t.Fatal("action did not stop")
		}
		select {
		case <-impl.stopped:
		default:
			t.Fatal("returned before plugin cleanup")
		}
		response, err := provider.Suggestions(context.Background(), suggestions.Request{})
		require.NoError(t, err)
		assert.Equal(t, suggestions.Response{AvailableActions: []suggestions.Action{{ID: "wait"}}}, response)
	})
}

type cancellableSuggestions struct {
	started chan struct{}
	stopped chan struct{}
}

func (*cancellableSuggestions) Suggestions(context.Context, suggestions.Request) (suggestions.Response, error) {
	return suggestions.Response{AvailableActions: []suggestions.Action{{ID: "wait"}}}, nil
}

func (p *cancellableSuggestions) RunAction(
	ctx context.Context, _ suggestions.ActionID, _ suggestions.Request,
) (suggestions.ActionResponse, error) {
	close(p.started)
	defer close(p.stopped)
	<-ctx.Done()
	return suggestions.ActionResponse{}, ctx.Err()
}

type observationConsumer struct {
	requests chan suggestions.Request
}

func (p observationConsumer) Suggestions(_ context.Context, request suggestions.Request) (suggestions.Response, error) {
	p.requests <- request
	var data struct {
		CPUCount int64 `json:"cpu_count"`
	}
	if err := request.TargetObservations.Decode("arm.fancy.target", &data); err != nil {
		return suggestions.Response{}, err
	}
	return suggestions.Response{Suggestions: []suggestions.Suggestion{
		{Value: strconv.FormatInt(data.CPUCount, 10)},
	}}, nil
}

func (p observationConsumer) RunAction(
	_ context.Context, _ suggestions.ActionID, request suggestions.Request,
) (suggestions.ActionResponse, error) {
	p.requests <- request
	var count int64
	if err := request.TargetObservations.Decode("arm.fancy.target.cpu_count", &count); err != nil {
		return suggestions.ActionResponse{}, err
	}
	return suggestions.ActionResponse{Suggestions: []suggestions.Suggestion{
		{Value: strconv.FormatInt(count, 10)},
	}}, nil
}
