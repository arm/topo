package deploy_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/stretchr/testify/assert"
)

func TestPublishedAddress(t *testing.T) {
	t.Run("strips the container-side port mapping", func(t *testing.T) {
		got := deploy.PublishedAddress("0.0.0.0:8080->80/tcp", "myhost")

		assert.Equal(t, "myhost:8080", got)
	})

	t.Run("replaces 0.0.0.0 with the supplied hostname", func(t *testing.T) {
		got := deploy.PublishedAddress("0.0.0.0:8080", "myhost")

		assert.Equal(t, "myhost:8080", got)
	})

	t.Run("returns empty string when there are no published ports", func(t *testing.T) {
		got := deploy.PublishedAddress("", "myhost")

		assert.Equal(t, "", got)
	})

	t.Run("leaves addresses without 0.0.0.0 untouched", func(t *testing.T) {
		got := deploy.PublishedAddress("127.0.0.1:8080", "myhost")

		assert.Equal(t, "127.0.0.1:8080", got)
	})

	t.Run("handles multiple comma-separated mappings", func(t *testing.T) {
		got := deploy.PublishedAddress("0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp", "myhost")

		assert.Equal(t, "myhost:8080", got)
	})
}
