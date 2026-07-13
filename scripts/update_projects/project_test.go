package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProject(t *testing.T) {
	t.Run("reads parameters from x-topo parameters", func(t *testing.T) {
		source := GitHubSource{Repo: "Arm-Examples/topo-example", SHA: "main"}
		composeFile := strings.NewReader(`
x-topo:
  name: Hello World
  description: A friendly project
  features:
    - web
  parameters:
    username:
      description: User name
      required: true
      example: alice
`)

		got, err := NewProject(source, composeFile)

		require.NoError(t, err)
		want := map[string]Parameter{
			"username": {
				Description: "User name",
				Required:    true,
				Example:     "alice",
			},
		}
		assert.Equal(t, want, got.Parameters)
	})

	t.Run("reads deprecated args as parameters", func(t *testing.T) {
		source := GitHubSource{Repo: "Arm-Examples/topo-example", SHA: "main"}
		composeFile := strings.NewReader(`
x-topo:
  name: Hello World
  description: A friendly project
  args:
    username:
      description: User name
      required: true
      default: alice
`)

		got, err := NewProject(source, composeFile)

		require.NoError(t, err)
		want := map[string]Parameter{
			"username": {
				Description: "User name",
				Required:    true,
				Default:     "alice",
			},
		}
		assert.Equal(t, want, got.Parameters)
	})

	t.Run("prefers parameters over args", func(t *testing.T) {
		source := GitHubSource{Repo: "Arm-Examples/topo-example", SHA: "main"}
		composeFile := strings.NewReader(`
x-topo:
  name: Hello World
  description: A friendly project
  parameters:
    username:
      description: New name
  args:
    username:
      description: Old name
    token:
      description: Secret token
`)

		got, err := NewProject(source, composeFile)

		require.NoError(t, err)
		want := map[string]Parameter{
			"username": {
				Description: "New name",
			},
		}
		assert.Equal(t, want, got.Parameters)
	})

	t.Run("reads parameters aliased to args", func(t *testing.T) {
		source := GitHubSource{Repo: "Arm-Examples/topo-example", SHA: "main"}
		composeFile := strings.NewReader(`
x-topo:
  name: Hello World
  description: A friendly project
  args: &args
    username:
      description: User name
      required: true
      example: alice
  parameters: *args
`)

		got, err := NewProject(source, composeFile)

		require.NoError(t, err)
		want := map[string]Parameter{
			"username": {
				Description: "User name",
				Required:    true,
				Example:     "alice",
			},
		}
		assert.Equal(t, want, got.Parameters)
	})
}
