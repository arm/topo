package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogSchema(t *testing.T) {
	t.Run("SchemaURL", func(t *testing.T) {
		t.Run("returns catalog schema URL", func(t *testing.T) {
			validator, err := NewCatalogSchema()
			require.NoError(t, err)

			got := validator.SchemaURL()

			assert.Equal(t, catalogSchemaURL, got)
		})
	})

	t.Run("ValidateTemplate", func(t *testing.T) {
		t.Run("accepts template that matches catalog schema", func(t *testing.T) {
			validator, err := NewCatalogSchema()
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

		t.Run("rejects template that does not match catalog schema", func(t *testing.T) {
			validator, err := NewCatalogSchema()
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
	})

	t.Run("ValidateCatalog", func(t *testing.T) {
		t.Run("accepts document that matches catalog schema", func(t *testing.T) {
			validator, err := NewCatalogSchema()
			require.NoError(t, err)
			document := Catalog{
				Schema: catalogSchemaURL,
				Templates: []Template{
					{
						XTopo: XTopo{
							Name:        "Hello World",
							Description: "A friendly template",
						},
						URL: "https://github.com/Arm-Examples/topo-welcome.git",
						Ref: "main",
					},
				},
			}

			err = validator.ValidateCatalog(document)

			assert.NoError(t, err)
		})

		t.Run("rejects document that does not match catalog schema", func(t *testing.T) {
			validator, err := NewCatalogSchema()
			require.NoError(t, err)
			document := Catalog{
				Schema: "https://example.com/catalog.schema.json",
				Templates: []Template{
					{
						XTopo: XTopo{
							Name:        "Hello World",
							Description: "A friendly template",
						},
						URL: "https://github.com/Arm-Examples/topo-welcome.git",
						Ref: "main",
					},
				},
			}

			err = validator.ValidateCatalog(document)

			assert.Error(t, err)
		})
	})
}
