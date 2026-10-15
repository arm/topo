package docker_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
