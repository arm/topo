package configs

import _ "embed"

//go:embed service-templates.json
var ServiceTemplatesJSON []byte

//go:embed example-projects.json
var ExampleProjectsJSON []byte
