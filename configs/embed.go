package configs

import _ "embed"

//go:embed service-templates.json
var ServiceTemplatesJSON []byte

//go:embed config-metadata.json
var ConfigMetadataJSON []byte

//go:embed Makefile-template.mk
var MakefileTemplate []byte

//go:embed version.txt
var VersionTxt string
