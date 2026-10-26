package deploy_test

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullImages(t *testing.T) {
	t.Run("skips services that have a build context", func(t *testing.T) {
		composeRunner := &fakeComposeCommandRunner{}

		composeFilePath := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  locally-built:
    build:
      context: .
      dockerfile_inline: "FROM alpine:latest"
    image: this-image-does-not-exist-on-docker-hub
  to-pull:
    image: whatever
`)
		err := deploy.PullImages(
			t.Context(),
			term.NewProgress(io.Discard),
			project.Scope{ComposeFile: composeFilePath},
			composeRunner,
		)

		require.NoError(t, err)
		want := []string{"pull", "to-pull"}
		assert.Equal(t, want, composeRunner.calls)
	})
}

func TestLogServices(t *testing.T) {
	t.Run("follows logs and delimits service names", func(t *testing.T) {
		runner := &fakeComposeCommandRunner{}
		options := deploy.LogsOptions{
			Services: []string{"web", "worker", "--follow"},
			Follow:   true,
		}

		err := deploy.LogServices(t.Context(), io.Discard, project.Scope{}, runner, options)

		require.NoError(t, err)
		assert.Equal(t, []string{"logs", "--tail", "100", "--follow", "--", "web", "worker", "--follow"}, runner.calls)
	})
}

func TestListContainersRaw(t *testing.T) {
	t.Run("propagates command construction errors", func(t *testing.T) {
		commandError := errors.New("cannot configure Compose")
		runner := &fakeComposeCommandRunner{commandError: commandError}

		got, err := deploy.ListContainersRaw(t.Context(), project.Scope{ComposeFile: "compose.yaml"}, runner, false)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, commandError)
	})

	t.Run("passes --all when all is enabled", func(t *testing.T) {
		runner := &fakeComposeCommandRunner{command: testutil.CmdWithOutput("", 0)}

		_, err := deploy.ListContainersRaw(t.Context(), project.Scope{ComposeFile: "compose.yaml"}, runner, true)

		require.NoError(t, err)
		assert.Equal(t, []string{"ps", "--format", "json", "--all"}, runner.calls)
	})

	t.Run("decodes the NDJSON stream emitted by Compose", func(t *testing.T) {
		input := `{"ID":"abc123","Names":"project-web-1","Image":"web","State":"running","Status":"Up 5 minutes","Ports":"0.0.0.0:8080->80/tcp"}
{"ID":"def456","Names":"project-db-1","Image":"db","State":"running","Status":"Up 5 minutes","Ports":""}`
		runner := &fakeComposeCommandRunner{command: testutil.CmdWithOutput(input, 0)}

		got, err := deploy.ListContainersRaw(t.Context(), project.Scope{ComposeFile: "compose.yaml"}, runner, false)

		require.NoError(t, err)
		want := []deploy.RawContainer{
			{
				ID: "abc123", Names: "project-web-1", Image: "web", State: "running",
				Status: "Up 5 minutes", Ports: "0.0.0.0:8080->80/tcp",
			},
			{
				ID: "def456", Names: "project-db-1", Image: "db", State: "running",
				Status: "Up 5 minutes", Ports: "",
			},
		}
		assert.Equal(t, want, got)
		assert.Equal(t, []string{"ps", "--format", "json"}, runner.calls)
	})
}

type fakeComposeCommandRunner struct {
	calls        []string
	command      *exec.Cmd
	commandError error
}

func (runner *fakeComposeCommandRunner) ComposeCommand(
	_ context.Context, _ project.Scope, args ...string,
) (*exec.Cmd, error) {
	runner.calls = args
	return runner.command, runner.commandError
}

func (runner *fakeComposeCommandRunner) RunComposeCommand(
	_ context.Context, _ io.Writer, _ project.Scope, args ...string,
) error {
	runner.calls = args
	return nil
}
