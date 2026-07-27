package fetch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arm/topo/internal/fetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Run("returns response body for successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusAccepted)
			_, _ = response.Write([]byte("contents"))
		}))
		defer server.Close()

		got, err := fetch.Get(context.Background(), server.URL)

		require.NoError(t, err)
		assert.Equal(t, []byte("contents"), got)
	})

	t.Run("returns error for unsuccessful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := fetch.Get(context.Background(), server.URL)

		assert.ErrorContains(t, err, "HTTP 404")
	})

	t.Run("rejects unsupported URL scheme", func(t *testing.T) {
		_, err := fetch.Get(context.Background(), "file:///tmp/catalog.json")

		assert.ErrorContains(t, err, "unsupported URL scheme")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := fetch.Get(ctx, "https://example.com")

		assert.ErrorIs(t, err, context.Canceled)
	})
}
