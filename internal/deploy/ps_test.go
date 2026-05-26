package deploy_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRunningContainers(t *testing.T) {
	t.Run("decodes the NDJSON stream emitted by docker compose ps", func(t *testing.T) {
		input := `{"Image":"web","Status":"Up 5 minutes","Ports":"0.0.0.0:8080->80/tcp"}
{"Image":"db","Status":"Up 5 minutes","Ports":""}`

		got, err := deploy.ParseRunningContainers(input)

		require.NoError(t, err)
		want := []deploy.RawContainer{
			{Image: "web", Status: "Up 5 minutes", Ports: "0.0.0.0:8080->80/tcp"},
			{Image: "db", Status: "Up 5 minutes", Ports: ""},
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns an empty slice for empty input", func(t *testing.T) {
		got, err := deploy.ParseRunningContainers("")

		require.NoError(t, err)
		assert.Equal(t, []deploy.RawContainer{}, got)
	})

	t.Run("returns an error on malformed JSON", func(t *testing.T) {
		_, err := deploy.ParseRunningContainers("{not json")

		assert.Error(t, err)
	})
}

func TestRemapAddresses(t *testing.T) {
	t.Run("strips the container-side port mapping and substitutes the hostname", func(t *testing.T) {
		input := []deploy.RawContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Address: "myhost:8080"}}
		assert.Equal(t, want, got)
	})

	t.Run("retains image and status fields", func(t *testing.T) {
		input := []deploy.RawContainer{{Image: "web", Status: "Up", Ports: ""}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Image: "web", Status: "Up"}}
		assert.Equal(t, want, got)
	})

	t.Run("leaves ports untouched when hostname is empty", func(t *testing.T) {
		input := []deploy.RawContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

		got := deploy.RemapAddresses(input, "")

		want := []deploy.Container{{Address: "0.0.0.0:8080->80/tcp"}}
		assert.Equal(t, want, got)
	})

	t.Run("leaves addresses without 0.0.0.0 untouched", func(t *testing.T) {
		input := []deploy.RawContainer{{Ports: "127.0.0.1:8080"}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Address: "127.0.0.1:8080"}}
		assert.Equal(t, want, got)
	})

	t.Run("remaps all published ports", func(t *testing.T) {
		input := []deploy.RawContainer{{Ports: "0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp"}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Address: "myhost:8080, myhost:8443"}}
		assert.Equal(t, want, got)
	})

	t.Run("returns an empty slice when given no containers", func(t *testing.T) {
		got := deploy.RemapAddresses(nil, "myhost")

		assert.Empty(t, got)
	})
}
