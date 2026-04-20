package compose_test

import (
	"strings"
	"testing"

	"github.com/arm/topo/internal/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullableServices(t *testing.T) {
	t.Run("returns services without a build key", func(t *testing.T) {
		got, err := compose.PullableServices(strings.NewReader(`
services:
  redis:
    image: redis:7
  postgres:
    image: postgres:16
`))

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"redis", "postgres"}, got)
	})

	t.Run("excludes services with a build key", func(t *testing.T) {
		got, err := compose.PullableServices(strings.NewReader(`
services:
  app:
    build: .
    image: myapp
  redis:
    image: redis:7
`))

		require.NoError(t, err)
		assert.Equal(t, []string{"redis"}, got)
	})

	t.Run("returns empty slice when all services are buildable", func(t *testing.T) {
		got, err := compose.PullableServices(strings.NewReader(`
services:
  app:
    build: .
  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
`))

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		_, err := compose.PullableServices(strings.NewReader(`{invalid`))

		assert.Error(t, err)
	})
}
