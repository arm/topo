package parameter_test

import (
	"errors"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Provide(definitions []parameter.Definition) (parameter.Values, error) {
	call := m.Called(definitions)
	if call.Get(0) == nil {
		return nil, call.Error(1)
	}
	return call.Get(0).(parameter.Values), call.Error(1)
}

func TestStrictProviderChain(t *testing.T) {
	t.Run("collects from single provider", func(t *testing.T) {
		provider := &mockProvider{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
		}
		provider.On("Provide", definitions).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictProviderChain(provider)

		got, err := chain.Provide(definitions)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		provider.AssertExpectations(t)
	})

	t.Run("errors when required parameters are missing", func(t *testing.T) {
		provider := &mockProvider{}
		missing := parameter.Definition{Name: "GREETING", Required: true, Description: "The greeting"}
		definitions := []parameter.Definition{
			missing,
			{Name: "PORT", Required: false},
		}
		provider.On("Provide", definitions).Return(parameter.Values{"PORT": "8080"}, nil)
		chain := parameter.NewStrictProviderChain(provider)

		_, err := chain.Provide(definitions)

		assert.Equal(t, parameter.MissingParametersError{missing}, err)
		provider.AssertExpectations(t)
	})

	t.Run("allows missing optional parameters", func(t *testing.T) {
		provider := &mockProvider{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider.On("Provide", definitions).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictProviderChain(provider)

		got, err := chain.Provide(definitions)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		provider.AssertExpectations(t)
	})

	t.Run("errors when provider fails", func(t *testing.T) {
		provider := &mockProvider{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
		}
		provider.On("Provide", mock.Anything).Return(nil, errors.New("big bang"))
		chain := parameter.NewStrictProviderChain(provider)

		_, err := chain.Provide(definitions)

		require.Error(t, err)
		assert.EqualError(t, err, "big bang")
		provider.AssertExpectations(t)
	})

	t.Run("stops calling providers when all required parameters are satisfied", func(t *testing.T) {
		provider1 := &mockProvider{}
		provider2 := &mockProvider{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		provider1.On("Provide", definitions).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictProviderChain(provider1, provider2)

		got, err := chain.Provide(definitions)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		provider1.AssertExpectations(t)
		provider2.AssertNotCalled(t, "Provide")
	})

	t.Run("calls second provider when first does not satisfy all required parameters", func(t *testing.T) {
		provider1 := &mockProvider{}
		provider2 := &mockProvider{}
		all := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "NAME", Required: true},
			{Name: "PORT", Required: false},
		}
		remaining := []parameter.Definition{
			{Name: "NAME", Required: true},
			{Name: "PORT", Required: false},
		}
		provider1.On("Provide", all).Return(parameter.Values{"GREETING": "Hello"}, nil)
		provider2.On("Provide", remaining).Return(parameter.Values{"NAME": "World"}, nil)
		chain := parameter.NewStrictProviderChain(provider1, provider2)

		got, err := chain.Provide(all)

		require.NoError(t, err)
		want := parameter.Values{
			"GREETING": "Hello",
			"NAME":     "World",
		}
		assert.Equal(t, want, got)
		provider1.AssertExpectations(t)
		provider2.AssertExpectations(t)
	})

	t.Run("collects values from multiple providers", func(t *testing.T) {
		provider1 := parameter.NewStaticProvider(parameter.Values{
			"PORT": "8080",
			"NAME": "Topo",
		})
		provider2 := parameter.NewStaticProvider(parameter.Values{
			"GREETING": "Hello",
		})
		chain := parameter.NewStrictProviderChain(provider1, provider2)
		definitions := []parameter.Definition{
			{Name: "NAME", Required: true},
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: true},
		}

		got, err := chain.Provide(definitions)

		require.NoError(t, err)
		want := parameter.Values{
			"NAME":     "Topo",
			"GREETING": "Hello",
			"PORT":     "8080",
		}
		assert.Equal(t, want, got)
	})

	t.Run("allows required parameters with non-empty current values", func(t *testing.T) {
		provider := parameter.NewStaticProvider(nil)
		chain := parameter.NewStrictProviderChain(provider)
		definitions := []parameter.Definition{{
			Name:          "CINNAMON",
			Required:      true,
			CurrentValues: []string{"current", "${CINNAMON}"},
		}}

		got, err := chain.Provide(definitions)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("errors when any current value is empty", func(t *testing.T) {
		provider := parameter.NewStaticProvider(nil)
		chain := parameter.NewStrictProviderChain(provider)
		definition := parameter.Definition{
			Name:          "CINNAMON",
			Required:      true,
			CurrentValues: []string{"current", ""},
		}

		_, err := chain.Provide([]parameter.Definition{definition})

		assert.Equal(t, parameter.MissingParametersError{definition}, err)
	})

	t.Run("does not provide omitted optional parameters", func(t *testing.T) {
		provider := parameter.NewStaticProvider(nil)
		chain := parameter.NewStrictProviderChain(provider)
		definitions := []parameter.Definition{
			{
				Name:     "CINNAMON",
				Required: false,
			},
		}

		got, err := chain.Provide(definitions)

		require.NoError(t, err)
		want := parameter.Values{}
		assert.Equal(t, want, got)
	})
}

func TestMissingParametersError(t *testing.T) {
	t.Run("formats error message with descriptions", func(t *testing.T) {
		err := parameter.MissingParametersError{
			{
				Name:        "GREETING",
				Description: "The greeting message",
				Example:     "Hello",
			},
			{
				Name:          "PORT",
				Description:   "Port number",
				CurrentValues: []string{"8080", ""},
			},
		}

		got := err.Error()

		want := `missing value(s) for required parameters:
  GREETING:
    description: The greeting message
    example: Hello
  PORT:
    description: Port number
    # current: ["8080",""]
`
		assert.Equal(t, want, got)
	})
}
