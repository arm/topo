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

func (m *mockProvider) Provide(args []arguments.Arg) ([]arguments.ResolvedArg, error) {
	callArgs := m.Called(args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).([]arguments.ResolvedArg), callArgs.Error(1)
}

func TestStrictMultiProvider(t *testing.T) {
	t.Run("collects from single provider", func(t *testing.T) {
		provider := &mockProvider{}
		args := []arguments.Arg{
			{Name: "GREETING", Required: true},
		}
		provider.On("Provide", args).Return([]arguments.ResolvedArg{{Name: "GREETING", Value: "Hello"}}, nil)
		multi := arguments.NewStrictProviderChain(provider)

		got, err := multi.Provide(args)

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
		provider.On("Provide", args).Return([]arguments.ResolvedArg{{Name: "PORT", Value: "8080"}}, nil)
		multi := arguments.NewStrictProviderChain(provider)

		_, err := multi.Provide(args)

		assert.Equal(t, arguments.MissingArgsError{missingArg}, err)
		provider.AssertExpectations(t)
	})

	t.Run("allows missing optional arguments", func(t *testing.T) {
		provider := &mockProvider{}
		args := []arguments.Arg{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider.On("Provide", args).Return([]arguments.ResolvedArg{{Name: "GREETING", Value: "Hello"}}, nil)
		multi := arguments.NewStrictProviderChain(provider)

		got, err := multi.Provide(args)

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
		provider.On("Provide", mock.Anything).Return(nil, errors.New("big bang"))
		multi := arguments.NewStrictProviderChain(provider)

		_, err := multi.Provide(args)

		require.Error(t, err)
		assert.EqualError(t, err, "big bang")
		provider.AssertExpectations(t)
	})

	t.Run("stops calling providers when all required args satisfied", func(t *testing.T) {
		provider1 := &mockProvider{}
		provider2 := &mockProvider{}
		args := []arguments.Arg{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider1.On("Provide", args).Return([]arguments.ResolvedArg{{Name: "GREETING", Value: "Hello"}}, nil)
		multi := arguments.NewStrictProviderChain(provider1, provider2)

		got, err := multi.Provide(args)

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
		provider1.On("Provide", allArgs).Return([]arguments.ResolvedArg{{Name: "GREETING", Value: "Hello"}}, nil)
		provider2.On("Provide", remainingArgs).Return([]arguments.ResolvedArg{{Name: "NAME", Value: "World"}}, nil)
		multi := arguments.NewStrictProviderChain(provider1, provider2)

		got, err := multi.Provide(allArgs)

		require.NoError(t, err)
		want := []arguments.ResolvedArg{
			{Name: "GREETING", Value: "Hello"},
			{Name: "NAME", Value: "World"},
		}
		assert.Equal(t, want, got)
		provider1.AssertExpectations(t)
		provider2.AssertExpectations(t)
	})

	t.Run("returns resolved args in requested order", func(t *testing.T) {
		provider1 := arguments.NewStaticProvider(
			arguments.ResolvedArg{Name: "PORT", Value: "8080"},
			arguments.ResolvedArg{Name: "NAME", Value: "Topo"},
		)
		provider2 := arguments.NewStaticProvider(
			arguments.ResolvedArg{Name: "GREETING", Value: "Hello"},
		)
		multi := arguments.NewStrictProviderChain(provider1, provider2)
		args := []arguments.Arg{
			{Name: "NAME", Required: true},
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: true},
		}

		got, err := multi.Provide(args)

		require.NoError(t, err)
		want := []arguments.ResolvedArg{
			{Name: "NAME", Value: "Topo"},
			{Name: "GREETING", Value: "Hello"},
			{Name: "PORT", Value: "8080"},
		}
		assert.Equal(t, want, got)
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
