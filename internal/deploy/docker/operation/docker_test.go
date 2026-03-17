package operation_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocker(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		testutil.RequireDocker(t)

		t.Run("executes docker command with args", func(t *testing.T) {
			var buf bytes.Buffer
			op := operation.NewDocker("Test docker version", ssh.PlainLocalhost, []string{"--version"})

			err := op.Run(&buf)

			require.NoError(t, err)
			assert.Contains(t, buf.String(), "Docker version")
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints command with multiple args and remote host", func(t *testing.T) {
			var buf bytes.Buffer
			remoteHost := ssh.Host("user@remote")
			op := operation.NewDocker("Test operation", remoteHost, []string{"ps", "-a", "--format", "json"})

			err := op.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf("docker -H %s ps -a --format json\n", remoteHost.AsURI())
			assert.Equal(t, want, got)
		})

		t.Run("prints command for localhost without host flag", func(t *testing.T) {
			var buf bytes.Buffer
			op := operation.NewDocker("Test operation", ssh.PlainLocalhost, []string{"images", "-q"})

			err := op.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := "docker images -q\n"
			assert.Equal(t, want, got)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("returns provided description", func(t *testing.T) {
			description := "Custom docker operation"
			op := operation.NewDocker(description, ssh.PlainLocalhost, []string{"info"})

			got := op.Description()

			assert.Equal(t, description, got)
		})
	})
}

func TestNewDockerPull(t *testing.T) {
	image := "nginx:latest"
	remoteHost := ssh.Host("user@remote")
	op := operation.NewDockerPull(remoteHost, image)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Pull image nginx:latest", got)
	})

	t.Run("DryRun", func(t *testing.T) {
		var buf bytes.Buffer

		err := op.DryRun(&buf)

		require.NoError(t, err)
		want := fmt.Sprintf("docker -H %s pull %s\n", remoteHost.AsURI(), image)
		assert.Equal(t, want, buf.String())
	})
}

func TestNewDockerStart(t *testing.T) {
	container := "my-container"
	remoteHost := ssh.Host("user@remote")
	op := operation.NewDockerStart(remoteHost, container)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Start container my-container", got)
	})

	t.Run("DryRun", func(t *testing.T) {
		var buf bytes.Buffer

		err := op.DryRun(&buf)

		require.NoError(t, err)
		want := fmt.Sprintf("docker -H %s start %s\n", remoteHost.AsURI(), container)
		assert.Equal(t, want, buf.String())
	})
}

func TestNewDockerRun(t *testing.T) {
	image := "alpine:latest"
	container := "test-container"
	remoteHost := ssh.Host("user@remote")

	t.Run("Description", func(t *testing.T) {
		op := operation.NewDockerRun(remoteHost, image, container, []string{"-d"})

		got := op.Description()

		assert.Equal(t, "Run image alpine:latest as container test-container", got)
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("with additional args", func(t *testing.T) {
			var buf bytes.Buffer
			op := operation.NewDockerRun(remoteHost, image, container, []string{"-d", "--restart", "always"})

			err := op.DryRun(&buf)

			require.NoError(t, err)
			want := fmt.Sprintf("docker -H %s run -d --restart always --name %s %s\n", remoteHost.AsURI(), container, image)
			assert.Equal(t, want, buf.String())
		})

		t.Run("with no additional args", func(t *testing.T) {
			var buf bytes.Buffer
			op := operation.NewDockerRun(remoteHost, image, container, []string{})

			err := op.DryRun(&buf)

			require.NoError(t, err)
			want := fmt.Sprintf("docker -H %s run --name %s %s\n", remoteHost.AsURI(), container, image)
			assert.Equal(t, want, buf.String())
		})
	})
}
