package configs

import _ "embed"

//go:embed templates.json
var TemplatesJSON []byte

//go:embed config-metadata.json
var ConfigMetadataJSON []byte

//go:embed Makefile-template.mk
var MakefileTemplate []byte

//go:embed version.txt
var VersionTxt string
