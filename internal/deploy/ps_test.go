package deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublishedAddress(t *testing.T) {
	t.Run("strips the container-side port mapping", func(t *testing.T) {
		got := publishedAddress("0.0.0.0:8080->80/tcp", "myhost")

		assert.Equal(t, "myhost:8080", got)
	})

	t.Run("replaces 0.0.0.0 with the supplied hostname", func(t *testing.T) {
		got := publishedAddress("0.0.0.0:8080", "myhost")

		assert.Equal(t, "myhost:8080", got)
	})

	t.Run("returns empty string when there are no published ports", func(t *testing.T) {
		got := publishedAddress("", "myhost")

		assert.Equal(t, "", got)
	})

	t.Run("leaves addresses without 0.0.0.0 untouched", func(t *testing.T) {
		got := publishedAddress("127.0.0.1:8080", "myhost")

		assert.Equal(t, "127.0.0.1:8080", got)
	})

	t.Run("keeps only the first mapping when several are published", func(t *testing.T) {
		got := publishedAddress("0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp", "myhost")

		assert.Equal(t, "myhost:8080", got)
	})
}
