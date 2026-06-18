package deploy_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContainers(t *testing.T) {
	t.Run("decodes the NDJSON stream emitted by docker compose ps", func(t *testing.T) {
		input := `{"ID":"abc123","Image":"web","Status":"Up 5 minutes","Ports":"0.0.0.0:8080->80/tcp"}
{"ID":"def456","Image":"db","Status":"Up 5 minutes","Ports":""}`

		got, err := deploy.ParseContainers(input)

		require.NoError(t, err)
		want := []deploy.PSContainer{
			{ID: "abc123", Image: "web", Status: "Up 5 minutes", Ports: "0.0.0.0:8080->80/tcp"},
			{ID: "def456", Image: "db", Status: "Up 5 minutes", Ports: ""},
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns an empty slice for empty input", func(t *testing.T) {
		got, err := deploy.ParseContainers("")

		require.NoError(t, err)
		assert.Equal(t, []deploy.PSContainer{}, got)
	})

	t.Run("returns an error on malformed JSON", func(t *testing.T) {
		_, err := deploy.ParseContainers("{not json")

		assert.Error(t, err)
	})
}

func TestParseInspectedContainers(t *testing.T) {
	t.Run("decodes docker inspect output", func(t *testing.T) {
		input := `[
			{
				"Id": "abc123",
				"Name": "/project-web-1",
				"HostConfig": {
					"Runtime": "io.containerd.remoteproc.v1",
					"Annotations": {
						"remoteproc.name": "m0"
					}
				}
			}
		]`

		got, err := deploy.ParseInspectedContainers(input)

		require.NoError(t, err)
		want := []deploy.InspectedContainer{
			{
				ID:   "abc123",
				Name: "/project-web-1",
				HostConfig: deploy.InspectedHostConfig{
					Runtime: "io.containerd.remoteproc.v1",
					Annotations: map[string]string{
						"remoteproc.name": "m0",
					},
				},
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		got, err := deploy.ParseInspectedContainers("")

		require.NoError(t, err)
		assert.Equal(t, []deploy.InspectedContainer{}, got)
	})

	t.Run("returns error on malformed JSON", func(t *testing.T) {
		_, err := deploy.ParseInspectedContainers("{not json")

		assert.Error(t, err)
	})
}

func TestBuildProcessingDomainLookup(t *testing.T) {
	t.Run("indexes remoteproc containers by id", func(t *testing.T) {
		input := []deploy.InspectedContainer{
			{
				ID: "abc123",
				HostConfig: deploy.InspectedHostConfig{
					Runtime: "io.containerd.remoteproc.v1",
					Annotations: map[string]string{
						"remoteproc.name": "m0",
					},
				},
			},
		}

		got := deploy.BuildProcessingDomainLookup(input)

		want := map[string]string{"abc123": "m0"}
		assert.Equal(t, want, got)
	})

	t.Run("indexes remoteproc containers by short docker id", func(t *testing.T) {
		input := []deploy.InspectedContainer{
			{
				ID: "0ccbbb5c98961db4e650d8db70b0154c3cf5bbfcd4a44450b019fea11a6afb9a",
				HostConfig: deploy.InspectedHostConfig{
					Runtime: "io.containerd.remoteproc.v1",
					Annotations: map[string]string{
						"remoteproc.name": "m33",
					},
				},
			},
		}

		got := deploy.BuildProcessingDomainLookup(input)

		want := map[string]string{
			"0ccbbb5c9896": "m33",
		}
		assert.Equal(t, want, got)
	})

	t.Run("skips non remoteproc containers", func(t *testing.T) {
		input := []deploy.InspectedContainer{
			{
				ID: "abc123",
				HostConfig: deploy.InspectedHostConfig{
					Runtime: "runc",
					Annotations: map[string]string{
						"remoteproc.name": "m0",
					},
				},
			},
		}

		got := deploy.BuildProcessingDomainLookup(input)

		assert.Empty(t, got)
	})

	t.Run("skips remoteproc containers without a processing domain annotation", func(t *testing.T) {
		input := []deploy.InspectedContainer{
			{
				ID: "abc123",
				HostConfig: deploy.InspectedHostConfig{
					Runtime:     "io.containerd.remoteproc.v1",
					Annotations: map[string]string{},
				},
			},
		}

		got := deploy.BuildProcessingDomainLookup(input)

		assert.Empty(t, got)
	})
}

func TestRemapAddresses(t *testing.T) {
	t.Run("strips the container-side port mapping and substitutes the hostname", func(t *testing.T) {
		input := []deploy.PSContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Address: "myhost:8080"}}
		assert.Equal(t, want, got)
	})

	t.Run("retains unmapped fields", func(t *testing.T) {
		input := []deploy.PSContainer{{Image: "web", Status: "Up", ID: "such-a-cool-id"}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Image: "web", Status: "Up", Id: "such-a-cool-id"}}
		assert.Equal(t, want, got)
	})

	t.Run("leaves ports untouched when hostname is empty", func(t *testing.T) {
		input := []deploy.PSContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

		got := deploy.RemapAddresses(input, "")

		want := []deploy.Container{{Address: "0.0.0.0:8080->80/tcp"}}
		assert.Equal(t, want, got)
	})

	t.Run("leaves addresses without 0.0.0.0 untouched", func(t *testing.T) {
		input := []deploy.PSContainer{{Ports: "127.0.0.1:8080"}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Address: "127.0.0.1:8080"}}
		assert.Equal(t, want, got)
	})

	t.Run("remaps all published ports", func(t *testing.T) {
		input := []deploy.PSContainer{{Ports: "0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp"}}

		got := deploy.RemapAddresses(input, "myhost")

		want := []deploy.Container{{Address: "myhost:8080, myhost:8443"}}
		assert.Equal(t, want, got)
	})

	t.Run("returns an empty slice when given no containers", func(t *testing.T) {
		got := deploy.RemapAddresses(nil, "myhost")

		assert.Empty(t, got)
	})
}
