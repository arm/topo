// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    catalogDocument, err := UnmarshalCatalogDocument(bytes)
//    bytes, err = catalogDocument.Marshal()

package catalog

import "encoding/json"

func UnmarshalCatalogDocument(data []byte) (CatalogDocument, error) {
	var r CatalogDocument
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CatalogDocument) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// A catalog of Topo Project repositories.
type CatalogDocument struct {
	// Schema URL for editor and tooling discovery.
	Schema   *Schema          `json:"$schema,omitempty"`
	Projects []ProjectElement `json:"projects"`
}

type ProjectElement struct {
	Description string `json:"description"`
	// Optional list of hardware, runtime, or platform features used by the project.
	Features []string `json:"features"`
	Name     string   `json:"name"`
	// Optional dictionary of parameter definitions used for parameterized projects.
	Parameters map[string]ParameterValue `json:"parameters,omitempty"`
	// Git ref to use when fetching the project repository.
	Ref string `json:"ref"`
	// Repository URL containing the project. This URL is the stable source identifier used to
	// match catalog entries to configured sources.
	URL string `json:"url"`
}

// Parameter definition.
type ParameterValue struct {
	// Value used if user skips input (only valid when not required).
	Default *string `json:"default,omitempty"`
	// Context displayed in user prompts.
	Description *string `json:"description,omitempty"`
	// Hint text displayed in help and prompts.
	Example *string `json:"example,omitempty"`
	// Advisory metadata that implementations may use to discover, filter, or suggest suitable
	// parameter values. Unknown hint keys should be ignored.
	Hints *Hints `json:"hints,omitempty"`
	// If true, implementations must enforce input or error.
	Required *bool `json:"required,omitempty"`
}

// Advisory metadata that implementations may use to discover, filter, or suggest suitable
// parameter values. Unknown hint keys should be ignored.
type Hints struct {
}

type Schema string

const (
	HTTPSRawGithubusercontentCOMArmTopoProjectCatalogMainDataCatalogSchemaJSON Schema = "https://raw.githubusercontent.com/arm/topo-project-catalog/main/data/catalog.schema.json"
)

const CatalogMajorVersion = "v1"
