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

func (m *mockResolver) Resolve(parameters []parameter.Parameter) (parameter.Values, error) {
	call := m.Called(parameters)
	if call.Get(0) == nil {
		return nil, call.Error(1)
	}
	return call.Get(0).(parameter.Values), call.Error(1)
}

func TestStrictResolverChain(t *testing.T) {
	t.Run("collects from single resolver", func(t *testing.T) {
		resolver := &mockResolver{}
		parameters := []parameter.Parameter{
			{Name: "GREETING", Required: true},
		}
		resolver.On("Resolve", parameters).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver.AssertExpectations(t)
	})

	t.Run("errors when required parameters are missing", func(t *testing.T) {
		resolver := &mockResolver{}
		missing := parameter.Parameter{Name: "GREETING", Description: "The greeting", Required: true}
		parameters := []parameter.Parameter{
			missing,
			{Name: "PORT", Required: false},
		}
		resolver.On("Resolve", parameters).Return(parameter.Values{"PORT": "8080"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		_, err := chain.Resolve(parameters)

		assert.Equal(t, parameter.MissingParametersError{missing}, err)
		resolver.AssertExpectations(t)
	})

	t.Run("allows missing optional parameters", func(t *testing.T) {
		resolver := &mockResolver{}
		parameters := []parameter.Parameter{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		resolver.On("Resolve", parameters).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver.AssertExpectations(t)
	})

	t.Run("errors when resolver fails", func(t *testing.T) {
		resolver := &mockResolver{}
		parameters := []parameter.Parameter{
			{Name: "GREETING", Required: true},
		}
		resolver.On("Resolve", mock.Anything).Return(nil, errors.New("big bang"))
		chain := parameter.NewStrictResolverChain(resolver)

		_, err := chain.Resolve(parameters)

		require.Error(t, err)
		assert.EqualError(t, err, "big bang")
		resolver.AssertExpectations(t)
	})

	t.Run("stops calling resolvers when all parameters are supplied", func(t *testing.T) {
		resolver1 := &mockResolver{}
		resolver2 := &mockResolver{}
		parameters := []parameter.Parameter{
			{Name: "GREETING", Required: true},
			{Name: "PORT", Required: false},
		}
		resolver1.On("Resolve", parameters).Return(parameter.Values{"GREETING": "Hello", "PORT": "8080"}, nil)
		chain := parameter.NewStrictResolverChain(resolver1, resolver2)

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello", "PORT": "8080"}
		assert.Equal(t, want, got)
		resolver1.AssertExpectations(t)
		resolver2.AssertNotCalled(t, "Resolve")
	})

	t.Run("passes only unsupplied parameters to the next resolver regardless of requiredness", func(t *testing.T) {
		resolver1 := &mockResolver{}
		resolver2 := &mockResolver{}
		all := []parameter.Parameter{
			{Name: "GREETING", Required: true},
			{Name: "NAME", Required: false},
			{Name: "PORT", ExistingValue: new("8080"), Required: false},
		}
		remaining := []parameter.Parameter{
			{Name: "NAME", Required: false},
			{Name: "PORT", ExistingValue: new("8080"), Required: false},
		}
		resolver1.On("Resolve", all).Return(parameter.Values{"GREETING": "Hello"}, nil)
		resolver2.On("Resolve", remaining).Return(parameter.Values{"NAME": "World"}, nil)
		chain := parameter.NewStrictResolverChain(resolver1, resolver2)

		got, err := chain.Resolve(all)

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
		parameters := []parameter.Parameter{{Name: "PORT", ExistingValue: new("8080"), Required: true}}

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("allows required parameters with empty current values", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(nil))
		parameters := []parameter.Parameter{{Name: "PORT", ExistingValue: new(""), Required: true}}

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("empty update overrides a non-empty current value", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(parameter.Values{"PORT": ""}))
		parameters := []parameter.Parameter{{Name: "PORT", ExistingValue: new("8080"), Required: true}}

		got, err := chain.Resolve(parameters)

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
