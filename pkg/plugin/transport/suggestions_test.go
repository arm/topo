package transport_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestSuggestionsRPCPlugin(t *testing.T) {
	t.Run("Suggestions", func(t *testing.T) {
		t.Run("round trips a suggestion request and response", func(t *testing.T) {
			requests := make(chan suggestions.Request, 1)
			serverProvider := roundTripSuggestions{requests: requests}
			serverAdapter := &transport.SuggestionsRPCPlugin{Impl: serverProvider}
			client := connectSuggestionsOverInMemoryGRPC(t, serverAdapter)
			request := suggestions.Request{
				Project:      suggestions.ProjectMetadata{Name: "Image segmentation"},
				Hints:        suggestions.Hints{"model": json.RawMessage(`{"format":"onnx"}`)},
				CurrentValue: new("existing model"),
			}

			got, err := client.Suggestions(t.Context(), request)

			require.NoError(t, err)
			want := suggestions.Response{
				Suggestions: []suggestions.Suggestion{{Value: "repo-1", Label: "Default model"}},
				AvailableActions: []suggestions.Action{
					{ID: "browse", Label: "Browse", Description: "Find a model"},
				},
			}
			assert.Equal(t, want, got)
			require.Len(t, requests, 1)
			assert.Equal(t, request, <-requests)
		})
	})

	t.Run("RunAction", func(t *testing.T) {
		t.Run("round trips the selected action and its request", func(t *testing.T) {
			requests := make(chan suggestions.Request, 1)
			serverProvider := roundTripSuggestions{requests: requests}
			serverAdapter := &transport.SuggestionsRPCPlugin{Impl: serverProvider}
			client := connectSuggestionsOverInMemoryGRPC(t, serverAdapter)
			value := "edited model"
			request := suggestions.Request{
				Project:      suggestions.ProjectMetadata{Name: "Image segmentation"},
				Hints:        suggestions.Hints{"model": json.RawMessage(`{"format":"onnx"}`)},
				CurrentValue: &value,
			}

			got, err := client.RunAction(t.Context(), "browse", request)

			require.NoError(t, err)
			want := suggestions.ActionResponse{Suggestions: []suggestions.Suggestion{
				{Value: "repo-2", Label: "browse"},
			}}
			assert.Equal(t, want, got)
			require.Len(t, requests, 1)
			assert.Equal(t, request, <-requests)
		})
	})
}

func connectSuggestionsOverInMemoryGRPC(
	t *testing.T, serverAdapter *transport.SuggestionsRPCPlugin,
) suggestions.Provider {
	t.Helper()
	server := grpc.NewServer()
	require.NoError(t, serverAdapter.GRPCServer(nil, server))
	listener := bufconn.Listen(1024 * 1024)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///plugin",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })
	clientAdapter := &transport.SuggestionsRPCPlugin{}
	client, err := clientAdapter.GRPCClient(t.Context(), nil, conn)
	require.NoError(t, err)
	return client.(suggestions.Provider)
}

type roundTripSuggestions struct {
	requests chan suggestions.Request
}

func (p roundTripSuggestions) Suggestions(_ context.Context, request suggestions.Request) (suggestions.Response, error) {
	p.requests <- request
	return suggestions.Response{
		Suggestions:      []suggestions.Suggestion{{Value: "repo-1", Label: "Default model"}},
		AvailableActions: []suggestions.Action{{ID: "browse", Label: "Browse", Description: "Find a model"}},
	}, nil
}

func (p roundTripSuggestions) RunAction(
	_ context.Context, id suggestions.ActionID, request suggestions.Request,
) (suggestions.ActionResponse, error) {
	p.requests <- request
	return suggestions.ActionResponse{Suggestions: []suggestions.Suggestion{{Value: "repo-2", Label: string(id)}}}, nil
}
