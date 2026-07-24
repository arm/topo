package docker_test

import (
	"testing"

	docker "github.com/arm/topo/internal/deploy/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContainers(t *testing.T) {
	t.Run("decodes the NDJSON stream emitted by docker compose ps", func(t *testing.T) {
		input := `{"ID":"abc123","Names":"project-web-1","Image":"web","State":"running","Status":"Up 5 minutes","Ports":"0.0.0.0:8080->80/tcp"}
{"ID":"def456","Names":"project-db-1","Image":"db","State":"running","Status":"Up 5 minutes","Ports":""}`

		got, err := docker.ParseContainers(input)

		require.NoError(t, err)
		want := []docker.PSContainer{
			{ID: "abc123", Names: "project-web-1", Image: "web", State: "running", Status: "Up 5 minutes", Ports: "0.0.0.0:8080->80/tcp"},
			{ID: "def456", Names: "project-db-1", Image: "db", State: "running", Status: "Up 5 minutes", Ports: ""},
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns an empty slice for empty input", func(t *testing.T) {
		got, err := docker.ParseContainers("")

		require.NoError(t, err)
		assert.Equal(t, []docker.PSContainer{}, got)
	})

	t.Run("returns an error on malformed JSON", func(t *testing.T) {
		_, err := docker.ParseContainers("{not json")

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

		got, err := docker.ParseInspectedContainers(input)

		require.NoError(t, err)
		want := []docker.InspectedContainer{
			{
				ID:   "abc123",
				Name: "/project-web-1",
				HostConfig: docker.InspectedHostConfig{
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
		got, err := docker.ParseInspectedContainers("")

		require.NoError(t, err)
		assert.Equal(t, []docker.InspectedContainer{}, got)
	})

	t.Run("returns error on malformed JSON", func(t *testing.T) {
		_, err := docker.ParseInspectedContainers("{not json")

		assert.Error(t, err)
	})
}

func TestBuildProcessingDomainLookup(t *testing.T) {
	t.Run("indexes remoteproc containers by id", func(t *testing.T) {
		input := []docker.InspectedContainer{
			{
				ID: "abc123",
				HostConfig: docker.InspectedHostConfig{
					Runtime: "io.containerd.remoteproc.v1",
					Annotations: map[string]string{
						"remoteproc.name": "m0",
					},
				},
			},
		}

		got := docker.BuildProcessingDomainLookup(input)

		want := map[string]string{"abc123": "m0"}
		assert.Equal(t, want, got)
	})

	t.Run("indexes remoteproc containers by short docker id", func(t *testing.T) {
		input := []docker.InspectedContainer{
			{
				ID: "0ccbbb5c98961db4e650d8db70b0154c3cf5bbfcd4a44450b019fea11a6afb9a",
				HostConfig: docker.InspectedHostConfig{
					Runtime: "io.containerd.remoteproc.v1",
					Annotations: map[string]string{
						"remoteproc.name": "m33",
					},
				},
			},
		}

		got := docker.BuildProcessingDomainLookup(input)

		want := map[string]string{
			"0ccbbb5c9896": "m33",
		}
		assert.Equal(t, want, got)
	})

	t.Run("skips non remoteproc containers", func(t *testing.T) {
		input := []docker.InspectedContainer{
			{
				ID: "abc123",
				HostConfig: docker.InspectedHostConfig{
					Runtime: "runc",
					Annotations: map[string]string{
						"remoteproc.name": "m0",
					},
				},
			},
		}

		got := docker.BuildProcessingDomainLookup(input)

		assert.Empty(t, got)
	})

	t.Run("skips remoteproc containers without a processing domain annotation", func(t *testing.T) {
		input := []docker.InspectedContainer{
			{
				ID: "abc123",
				HostConfig: docker.InspectedHostConfig{
					Runtime:     "io.containerd.remoteproc.v1",
					Annotations: map[string]string{},
				},
			},
		}

		got := docker.BuildProcessingDomainLookup(input)

		assert.Empty(t, got)
	})
}

func TestRemapAddresses(t *testing.T) {
	t.Run("strips the container-side port mapping and substitutes the hostname", func(t *testing.T) {
		input := []docker.PSContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

		got := docker.RemapAddresses(input, "myhost")

		want := []docker.Container{{Address: "myhost:8080"}}
		assert.Equal(t, want, got)
	})

	t.Run("retains unmapped fields", func(t *testing.T) {
		input := []docker.PSContainer{{Image: "web", State: "running", Status: "Up", ID: "such-a-cool-id", Names: "project-web-1"}}

		got := docker.RemapAddresses(input, "myhost")

		want := []docker.Container{{Image: "web", State: "running", Status: "Up", Id: "such-a-cool-id", Names: "project-web-1"}}
		assert.Equal(t, want, got)
	})

	t.Run("leaves ports untouched when hostname is empty", func(t *testing.T) {
		input := []docker.PSContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

		got := docker.RemapAddresses(input, "")

		want := []docker.Container{{Address: "0.0.0.0:8080->80/tcp"}}
		assert.Equal(t, want, got)
	})

	t.Run("leaves addresses without 0.0.0.0 untouched", func(t *testing.T) {
		input := []docker.PSContainer{{Ports: "127.0.0.1:8080"}}

		got := docker.RemapAddresses(input, "myhost")

		want := []docker.Container{{Address: "127.0.0.1:8080"}}
		assert.Equal(t, want, got)
	})

	t.Run("remaps all published ports", func(t *testing.T) {
		input := []docker.PSContainer{{Ports: "0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp"}}

		got := docker.RemapAddresses(input, "myhost")

		want := []docker.Container{{Address: "myhost:8080, myhost:8443"}}
		assert.Equal(t, want, got)
	})

	t.Run("returns an empty slice when given no containers", func(t *testing.T) {
		got := docker.RemapAddresses(nil, "myhost")

		assert.Empty(t, got)
	})
}
