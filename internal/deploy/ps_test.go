package deploy

import (
	"testing"

	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestPS(t *testing.T) {
	t.Run("rejects an unsupported engine", func(t *testing.T) {
		got, err := PS(
			t.Context(), project.Scope{}, ssh.PlainLocalhost, EngineKind("unknown"), "target", false,
		)

		assert.Nil(t, got)
		assert.ErrorContains(t, err, `invalid engine "unknown"`)
	})
}

func TestEnrichContainers(t *testing.T) {
	t.Run("processing domains", func(t *testing.T) {
		t.Run("assigns processing domains by ID and defaults unmatched containers to Linux Host", func(t *testing.T) {
			input := []RawContainer{{ID: "remote"}, {ID: "host"}}
			domains := map[string]string{"remote": "m33", "unlisted": "m0"}

			got := enrichContainers(input, "myhost", domains)

			want := []EnrichedContainer{
				{Id: "remote", ProcessingDomain: "m33"},
				{Id: "host", ProcessingDomain: "Linux Host"},
			}
			assert.Equal(t, want, got)
		})

		t.Run("defaults to Linux Host when the domain lookup is empty", func(t *testing.T) {
			input := []RawContainer{{ID: "host"}}

			got := enrichContainers(input, "myhost", map[string]string{})

			want := []EnrichedContainer{{Id: "host", ProcessingDomain: "Linux Host"}}
			assert.Equal(t, want, got)
		})
	})

	t.Run("port mapping", func(t *testing.T) {
		t.Run("strips the container-side port mapping and substitutes the hostname", func(t *testing.T) {
			input := []RawContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

			got := enrichContainers(input, "myhost", nil)

			want := []EnrichedContainer{{ProcessingDomain: "Linux Host", Address: "myhost:8080"}}
			assert.Equal(t, want, got)
		})

		t.Run("leaves ports untouched when hostname is empty", func(t *testing.T) {
			input := []RawContainer{{Ports: "0.0.0.0:8080->80/tcp"}}

			got := enrichContainers(input, "", nil)

			want := []EnrichedContainer{{ProcessingDomain: "Linux Host", Address: "0.0.0.0:8080->80/tcp"}}
			assert.Equal(t, want, got)
		})

		t.Run("leaves addresses without 0.0.0.0 untouched", func(t *testing.T) {
			input := []RawContainer{{Ports: "127.0.0.1:8080"}}

			got := enrichContainers(input, "myhost", nil)

			want := []EnrichedContainer{{ProcessingDomain: "Linux Host", Address: "127.0.0.1:8080"}}
			assert.Equal(t, want, got)
		})

		t.Run("remaps all published ports", func(t *testing.T) {
			input := []RawContainer{{Ports: "0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp"}}

			got := enrichContainers(input, "myhost", nil)

			want := []EnrichedContainer{{ProcessingDomain: "Linux Host", Address: "myhost:8080, myhost:8443"}}
			assert.Equal(t, want, got)
		})
	})

	t.Run("combines address mapping with processing domain enrichment", func(t *testing.T) {
		input := []RawContainer{{ID: "remote", Ports: "0.0.0.0:8080->80/tcp"}}
		domains := map[string]string{"remote": "m33"}

		got := enrichContainers(input, "myhost", domains)

		want := []EnrichedContainer{{Id: "remote", ProcessingDomain: "m33", Address: "myhost:8080"}}
		assert.Equal(t, want, got)
	})

	t.Run("retains unmapped fields", func(t *testing.T) {
		input := []RawContainer{{
			Image: "web", State: "running", Status: "Up", ID: "such-a-cool-id", Names: "project-web-1",
		}}

		got := enrichContainers(input, "myhost", nil)

		want := []EnrichedContainer{{
			ProcessingDomain: "Linux Host", Image: "web", State: "running", Status: "Up",
			Id: "such-a-cool-id", Names: "project-web-1",
		}}
		assert.Equal(t, want, got)
	})

	t.Run("returns an empty slice when given no containers", func(t *testing.T) {
		got := enrichContainers(nil, "myhost", nil)

		assert.Empty(t, got)
	})
}
