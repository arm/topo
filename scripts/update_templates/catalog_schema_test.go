package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogSchema(t *testing.T) {
	t.Run("ValidateTemplate accepts template that matches catalog schema", func(t *testing.T) {
		schemaJSON, err := readCatalogSchema()
		require.NoError(t, err)
		validator, err := NewCatalogSchemaFromBytes(schemaJSON)
		require.NoError(t, err)
		template := Template{
			XTopo: XTopo{
				Name:        "Hello World",
				Description: "A friendly template",
				Features:    []string{"web"},
			},
			URL: "https://github.com/Arm-Examples/topo-welcome.git",
			Ref: "main",
		}

		err = validator.ValidateTemplate(template)

		assert.NoError(t, err)
	})

	t.Run("ValidateTemplate rejects template that does not match catalog schema", func(t *testing.T) {
		schemaJSON, err := readCatalogSchema()
		require.NoError(t, err)
		validator, err := NewCatalogSchemaFromBytes(schemaJSON)
		require.NoError(t, err)
		template := Template{
			XTopo: XTopo{
				Description: "Missing a name",
			},
			URL: "https://github.com/Arm-Examples/topo-welcome.git",
			Ref: "main",
		}

		err = validator.ValidateTemplate(template)

		assert.Error(t, err)
	})
}
