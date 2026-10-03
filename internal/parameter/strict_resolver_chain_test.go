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
			{Name: "GREETING", References: []parameter.Reference{{Expression: "${GREETING:?required}"}}},
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
		missing := parameter.Parameter{Name: "GREETING", Description: "The greeting", References: []parameter.Reference{{Expression: "${GREETING:?required}"}}}
		parameters := []parameter.Parameter{
			missing,
			{Name: "PORT"},
		}
		resolver.On("Resolve", parameters).Return(parameter.Values{"PORT": "8080"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		_, err := chain.Resolve(parameters)

		assert.ErrorContains(t, err, "parameter validation failed")
		resolver.AssertExpectations(t)
	})

	t.Run("allows parameters without required interpolation", func(t *testing.T) {
		resolver := &mockResolver{}
		parameters := []parameter.Parameter{
			{Name: "GREETING", References: []parameter.Reference{{Expression: "${GREETING:?required}"}}},
			{Name: "PORT", References: []parameter.Reference{{Expression: "${PORT}"}}},
		}
		resolver.On("Resolve", parameters).Return(parameter.Values{"GREETING": "Hello"}, nil)
		chain := parameter.NewStrictResolverChain(resolver)

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		want := parameter.Values{"GREETING": "Hello"}
		assert.Equal(t, want, got)
		resolver.AssertExpectations(t)
	})

	t.Run("allows an unset parameter without any required references", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain()
		parameters := []parameter.Parameter{{
			Name:       "PORT",
			References: []parameter.Reference{{Expression: "${PORT:-8080}"}, {Expression: "${PORT:-9090}"}},
		}}

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("rejects an unset parameter with a required reference", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain()
		missing := parameter.Parameter{
			Name:       "PORT",
			References: []parameter.Reference{{Expression: "${PORT:-8080}"}, {Expression: "${PORT?required}"}},
		}

		_, err := chain.Resolve([]parameter.Parameter{missing})

		assert.ErrorContains(t, err, "parameter validation failed")
	})

	t.Run("errors when resolver fails", func(t *testing.T) {
		resolver := &mockResolver{}
		parameters := []parameter.Parameter{
			{Name: "GREETING"},
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
			{Name: "GREETING"},
			{Name: "PORT"},
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
			{Name: "GREETING", References: []parameter.Reference{{Expression: "${GREETING:?required}"}}},
			{Name: "NAME"},
			{Name: "PORT", ExistingValue: new("8080"), References: []parameter.Reference{{Expression: "${PORT}"}}},
		}
		remaining := []parameter.Parameter{
			{Name: "NAME"},
			{Name: "PORT", ExistingValue: new("8080"), References: []parameter.Reference{{Expression: "${PORT}"}}},
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
		parameters := []parameter.Parameter{{Name: "PORT", ExistingValue: new("8080"), References: []parameter.Reference{{Expression: "${PORT:?required}"}}}}

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("accepts an empty value for a presence-required parameter", func(t *testing.T) {
		values := parameter.Values{"PORT": ""}
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(values))
		parameters := []parameter.Parameter{{Name: "PORT", References: []parameter.Reference{{Expression: "${PORT?required}"}}}}

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, values, got)
	})

	t.Run("rejects an empty value for a value-required parameter", func(t *testing.T) {
		values := parameter.Values{"PORT": ""}
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(values))
		parameters := []parameter.Parameter{{Name: "PORT", References: []parameter.Reference{{Expression: "${PORT:?required}"}}}}

		got, err := chain.Resolve(parameters)

		require.Error(t, err)
		assert.Empty(t, got)
	})

	t.Run("empty update overrides a non-empty current value", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(parameter.Values{"PORT": ""}))
		parameters := []parameter.Parameter{{Name: "PORT", ExistingValue: new("8080")}}

		got, err := chain.Resolve(parameters)

		require.NoError(t, err)
		assert.Equal(t, parameter.Values{"PORT": ""}, got)
	})

	t.Run("formats validation error with paths and reasons", func(t *testing.T) {
		chain := parameter.NewStrictResolverChain(parameter.NewStaticResolver(nil))
		parameters := []parameter.Parameter{
			{
				Name:        "GREETING",
				Description: "greeting",
				References:  []parameter.Reference{{Expression: "${GREETING:?required}", Path: "a.b.c"}},
			},
			{
				Name:       "PORT",
				Example:    "example value",
				References: []parameter.Reference{{Expression: "${PORT:?required}", Path: "a.b"}},
			},
		}

		got, err := chain.Resolve(parameters)

		require.EqualError(t, err, `parameter validation failed:
GREETING:
  description: greeting
  references:
    - a.b.c: ${GREETING:?required}
  reason: required variable GREETING is missing a value
PORT:
  example: example value
  references:
    - a.b: ${PORT:?required}
  reason: required variable PORT is missing a value`)
		assert.Empty(t, got)
	})
}
