package parameter_test

import (
	"errors"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockResolver struct {
	mock.Mock
}

func (m *mockResolver) Resolve(definitions []parameter.Definition, currentValues parameter.Values) (parameter.Values, error) {
	call := m.Called(definitions, currentValues)
	if call.Get(0) == nil {
		return nil, call.Error(1)
	}
	return call.Get(0).(parameter.Values), call.Error(1)
}

func TestStrictResolverChain(t *testing.T) {
	t.Run("collects from single resolver", func(t *testing.T) {
		resolver := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
		}
		resolver.On("Resolve", definitions, parameter.Values(nil)).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		got, err := chain.Resolve(definitions, nil)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver.AssertExpectations(t)
	})

	t.Run("errors when required parameters are missing", func(t *testing.T) {
		resolver := &mockResolver{}
		missing := parameter.Definition{Name: "GREETING", Required: true, Description: "The greeting"}
		definitions := []parameter.Definition{
			missing,
			{Name: "PORT", Required: false},
		}
		resolver.On("Resolve", definitions, parameter.Values(nil)).Return(parameter.Values{"PORT": "8080"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		_, err := chain.Resolve(definitions, nil)

		assert.Equal(t, parameter.MissingParametersError{missing}, err)
		resolver.AssertExpectations(t)
	})

	t.Run("allows missing optional parameters", func(t *testing.T) {
		resolver := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		resolver.On("Resolve", definitions, parameter.Values(nil)).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		got, err := chain.Resolve(definitions, nil)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver.AssertExpectations(t)
	})

	t.Run("errors when resolver fails", func(t *testing.T) {
		resolver := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
		}
		resolver.On("Resolve", mock.Anything, parameter.Values(nil)).Return(nil, errors.New("big bang"))
		chain := parameter.NewStrictResolverChain(resolver)

		_, err := chain.Resolve(definitions, nil)

		require.Error(t, err)
		assert.EqualError(t, err, "big bang")
		resolver.AssertExpectations(t)
	})

	t.Run("stops calling resolvers when all parameters are supplied", func(t *testing.T) {
		resolver1 := &mockResolver{}
		resolver2 := &mockResolver{}
		definitions := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		resolver1.On("Resolve", definitions, parameter.Values(nil)).Return(parameter.Values{"GREETING": "Hello", "PORT": "8080"}, nil)
		chain := parameter.NewStrictResolverChain(resolver1, resolver2)

		got, err := chain.Resolve(definitions, nil)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello", "PORT": "8080"}
		assert.Equal(t, want, got)
		resolver1.AssertExpectations(t)
		resolver2.AssertNotCalled(t, "Resolve")
	})

	t.Run("passes only unsupplied parameters to the next resolver regardless of requiredness", func(t *testing.T) {
		resolver1 := &mockResolver{}
		resolver2 := &mockResolver{}
		all := []parameter.Definition{
			{Name: "GREETING", Required: true},
			{Name: "NAME", Required: false},
			{Name: "PORT", Required: false},
		}
		remaining := []parameter.Definition{
			{Name: "NAME", Required: false},
			{Name: "PORT", Required: false},
		}
		currentValues := parameter.Values{"PORT": "8080"}
		resolver1.On("Resolve", all, currentValues).Return(parameter.Values{"GREETING": "Hello"}, nil)
		resolver2.On("Resolve", remaining, currentValues).Return(parameter.Values{"NAME": "World"}, nil)
		chain := parameter.NewStrictResolverChain(resolver1, resolver2)

		got, err := chain.Resolve(all, currentValues)

		require.NoError(t, err)
		want := parameter.Values{
			"GREETING": "Hello",
			"NAME":     "World",
		}
		assert.Equal(t, want, got)
		resolver1.AssertExpectations(t)
		resolver2.AssertExpectations(t)
	})

	t.Run("allows required parameters with non-empty current values", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(nil))
		definitions := []parameter.Definition{{Name: "PORT", Required: true}}

		got, err := chain.Resolve(definitions, parameter.Values{"PORT": "8080"})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("allows required parameters with empty current values", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(nil))
		definitions := []parameter.Definition{{Name: "PORT", Required: true}}

		got, err := chain.Resolve(definitions, parameter.Values{"PORT": ""})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("empty update overrides a non-empty current value", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(parameter.Values{"PORT": ""}))
		definitions := []parameter.Definition{{Name: "PORT", Required: true}}

		got, err := chain.Resolve(definitions, parameter.Values{"PORT": "8080"})

		require.NoError(t, err)
		assert.Equal(t, parameter.Values{"PORT": ""}, got)
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
				Name:        "PORT",
				Description: "Port number",
			},
		}

		got := err.Error()

		want := `missing value(s) for required parameters:
  GREETING:
    description: The greeting message
    example: Hello
  PORT:
    description: Port number
`
		assert.Equal(t, want, got)
	})
}
