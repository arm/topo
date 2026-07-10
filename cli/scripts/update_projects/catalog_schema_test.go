package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogSchema(t *testing.T) {
	t.Run("SchemaURL", func(t *testing.T) {
		t.Run("returns catalog schema URL", func(t *testing.T) {
			schemaPath, err := CatalogSchemaFilePath()
			require.NoError(t, err)
			validator, err := NewCatalogSchema(schemaPath)
			require.NoError(t, err)

			got := validator.SchemaURL()

			assert.Equal(t, catalogSchemaURL, got)
		})
	})

	t.Run("ValidateProject", func(t *testing.T) {
		t.Run("accepts project that matches catalog schema", func(t *testing.T) {
			schemaPath, err := CatalogSchemaFilePath()
			require.NoError(t, err)
			validator, err := NewCatalogSchema(schemaPath)
			require.NoError(t, err)
			project := Project{
				XTopo: XTopo{
					Name:        "Hello World",
					Description: "A friendly project",
					Features:    []string{"web"},
				},
				URL: "https://github.com/Arm-Examples/topo-welcome.git",
				Ref: "main",
			}

			err = validator.ValidateProject(project)

			assert.NoError(t, err)
		})

		t.Run("rejects project that does not match catalog schema", func(t *testing.T) {
			schemaPath, err := CatalogSchemaFilePath()
			require.NoError(t, err)
			validator, err := NewCatalogSchema(schemaPath)
			require.NoError(t, err)
			project := Project{
				XTopo: XTopo{
					Description: "Missing a name",
				},
				URL: "https://github.com/Arm-Examples/topo-welcome.git",
				Ref: "main",
			}

			err = validator.ValidateProject(project)

			assert.Error(t, err)
		})
	})

	t.Run("ValidateCatalog", func(t *testing.T) {
		t.Run("accepts document that matches catalog schema", func(t *testing.T) {
			schemaPath, err := CatalogSchemaFilePath()
			require.NoError(t, err)
			validator, err := NewCatalogSchema(schemaPath)
			require.NoError(t, err)
			document := Catalog{
				Schema: catalogSchemaURL,
				Projects: []Project{
					{
						XTopo: XTopo{
							Name:        "Hello World",
							Description: "A friendly project",
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
			schemaPath, err := CatalogSchemaFilePath()
			require.NoError(t, err)
			validator, err := NewCatalogSchema(schemaPath)
			require.NoError(t, err)
			document := Catalog{
				Schema: "https://example.com/catalog.schema.json",
				Projects: []Project{
					{
						XTopo: XTopo{
							Name:        "Hello World",
							Description: "A friendly project",
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
