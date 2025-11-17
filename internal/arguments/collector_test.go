package arguments_test

import (
	"errors"
	"testing"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Provide(args []arguments.Arg) (map[string]string, error) {
	callArgs := m.Called(args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(map[string]string), callArgs.Error(1)
}

func (m *mockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func TestCollector(t *testing.T) {
	t.Run("collects from single provider", func(t *testing.T) {
		provider := &mockProvider{}
		args := []arguments.Arg{
			{Name: "GREETING", Required: true},
		}
		provider.On("Provide", args).Return(map[string]string{"GREETING": "Hello"}, nil)
		collector := arguments.NewCollector(provider)

		got, err := collector.Collect(args)

		require.NoError(t, err)
		want := []arguments.ResolvedArg{{Name: "GREETING", Value: "Hello"}}
		assert.Equal(t, want, got)
		provider.AssertExpectations(t)
	})

	t.Run("errors when required arguments missing", func(t *testing.T) {
		provider := &mockProvider{}
		missingArg := arguments.Arg{Name: "GREETING", Required: true, Description: "The greeting"}
		args := []arguments.Arg{
			missingArg,
			{Name: "PORT", Required: false},
		}
		provider.On("Provide", args).Return(map[string]string{"PORT": "8080"}, nil)
		collector := arguments.NewCollector(provider)

		_, err := collector.Collect(args)

		assert.Equal(t, arguments.MissingArgsError{missingArg}, err)
		provider.AssertExpectations(t)
	})

	t.Run("allows missing optional arguments", func(t *testing.T) {
		provider := &mockProvider{}
		args := []arguments.Arg{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider.On("Provide", args).Return(map[string]string{"GREETING": "Hello"}, nil)
		collector := arguments.NewCollector(provider)

		got, err := collector.Collect(args)

		require.NoError(t, err)
		want := []arguments.ResolvedArg{{Name: "GREETING", Value: "Hello"}}
		assert.Equal(t, want, got)
		provider.AssertExpectations(t)
	})

	t.Run("errors when provider fails", func(t *testing.T) {
		provider := &mockProvider{}
		args := []arguments.Arg{
			{Name: "GREETING", Required: true},
		}
		provider.On("Name").Return("fancy")
		provider.On("Provide", mock.Anything).Return(nil, errors.New("big bang"))
		collector := arguments.NewCollector(provider)

		_, err := collector.Collect(args)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fancy provider failed: big bang")
		provider.AssertExpectations(t)
	})

	t.Run("stops calling providers when all required args satisfied", func(t *testing.T) {
		provider1 := &mockProvider{}
		provider2 := &mockProvider{}
		args := []arguments.Arg{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider1.On("Provide", args).Return(map[string]string{"GREETING": "Hello"}, nil)
		collector := arguments.NewCollector(provider1, provider2)

		got, err := collector.Collect(args)

		require.NoError(t, err)
		want := []arguments.ResolvedArg{{Name: "GREETING", Value: "Hello"}}
		assert.Equal(t, want, got)
		provider1.AssertExpectations(t)
		provider2.AssertNotCalled(t, "Provide")
	})

	t.Run("calls second provider when first does not satisfy all required args", func(t *testing.T) {
		provider1 := &mockProvider{}
		provider2 := &mockProvider{}
		allArgs := []arguments.Arg{
			{Name: "GREETING", Required: true},
			{Name: "NAME", Required: true},
			{Name: "PORT", Required: false},
		}
		remainingArgs := []arguments.Arg{
			{Name: "NAME", Required: true},
			{Name: "PORT", Required: false},
		}
		provider1.On("Provide", allArgs).Return(map[string]string{"GREETING": "Hello"}, nil)
		provider2.On("Provide", remainingArgs).Return(map[string]string{"NAME": "World"}, nil)
		collector := arguments.NewCollector(provider1, provider2)

		got, err := collector.Collect(allArgs)

		require.NoError(t, err)
		want := []arguments.ResolvedArg{
			{Name: "GREETING", Value: "Hello"},
			{Name: "NAME", Value: "World"},
		}
		assert.Equal(t, want, got)
		provider1.AssertExpectations(t)
		provider2.AssertExpectations(t)
	})
}

func TestMissingArgsError(t *testing.T) {
	t.Run("formats error message with descriptions", func(t *testing.T) {
		err := arguments.MissingArgsError{
			{
				Name:        "GREETING",
				Description: "The greeting message",
				Example:     "Hello",
			},
			{
				Name:        "PORT",
				Description: "Port number",
			},
		}

		got := err.Error()

		want := `missing required build arguments:
  GREETING:
    description: The greeting message
    example: Hello
  PORT:
    description: Port number
`
		assert.Equal(t, want, got)
	})
}
