package parameter_test

import (
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/stretchr/testify/assert"
)

func TestParameterAssertSatisfiedBy(t *testing.T) {
	t.Run("ignores a missing required value for another parameter", func(t *testing.T) {
		p := parameter.Parameter{
			Name:       "PORT",
			References: []parameter.Reference{{Expression: "${HOST:?host is required}:${PORT}"}},
		}
		value := "8080"

		err := p.AssertSatisfiedBy(&value)

		assert.NoError(t, err)
	})

	t.Run("checks a required parameter after another parameter in the same string", func(t *testing.T) {
		p := parameter.Parameter{
			Name:       "PORT",
			References: []parameter.Reference{{Expression: "${HOST:?host is required}:${PORT:?port is required}"}},
		}

		err := p.AssertSatisfiedBy(nil)

		assert.ErrorContains(t, err, "required variable PORT")
	})

	t.Run("accepts a value satisfying all references", func(t *testing.T) {
		p := parameter.Parameter{
			Name: "PORT",
			References: []parameter.Reference{
				{Expression: "${PORT?port is required}"},
				{Expression: "${PORT:?port must not be empty}"},
			},
		}
		value := "8080"

		err := p.AssertSatisfiedBy(&value)

		assert.NoError(t, err)
	})

	t.Run("accepts an unset optional parameter", func(t *testing.T) {
		p := parameter.Parameter{
			Name: "PORT",
			References: []parameter.Reference{
				{Expression: "${PORT}"},
				{Expression: "${PORT:-8080}"},
			},
		}

		err := p.AssertSatisfiedBy(nil)

		assert.NoError(t, err)
	})

	t.Run("rejects an unset parameter required by a reference", func(t *testing.T) {
		p := parameter.Parameter{
			Name: "PORT",
			References: []parameter.Reference{
				{Expression: "${PORT:-8080}"},
				{Expression: "${PORT?port is required}"},
			},
		}

		err := p.AssertSatisfiedBy(nil)

		assert.ErrorContains(t, err, "required variable PORT")
	})

	t.Run("accepts an empty value when only presence is required", func(t *testing.T) {
		p := parameter.Parameter{
			Name:       "PORT",
			References: []parameter.Reference{{Expression: "${PORT?port is required}"}},
		}
		value := ""

		err := p.AssertSatisfiedBy(&value)

		assert.NoError(t, err)
	})

	t.Run("rejects an empty value when a nonempty value is required", func(t *testing.T) {
		p := parameter.Parameter{
			Name:       "PORT",
			References: []parameter.Reference{{Expression: "${PORT:?port must not be empty}"}},
		}
		value := ""

		err := p.AssertSatisfiedBy(&value)

		assert.ErrorContains(t, err, "required variable PORT")
	})
}
