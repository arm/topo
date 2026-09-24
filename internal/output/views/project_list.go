package views

import (
	"bytes"
	"text/template"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/term"
)

type ProjectList []catalog.ProjectWithCompatibility

const projectListTemplate = `
{{- define "projectRow" }}
{{- if .Compatibility }}{{ compatibilityMark .Compatibility }} {{ end }}{{ cyan .Name }}
  {{ blue "Clone:" }}
    {{ cloneCommand . }}
{{- if .Features }}
  {{ blue "Features:" }}
  {{- range .Features }}
    {{ . }}
  {{- end }}
{{- end }}
{{- if .Description }}

{{ wrap .Description }}
{{- end }}
{{- end }}

{{- define "projectList" }}
{{- range . }}
{{- template "projectRow" . }}

{{ end }}
{{- end }}`

func (r ProjectList) AsJSON() (string, error) {
	return asJSON(r)
}

func (r ProjectList) AsPlain(palette term.Palette) (string, error) {
	funcMap := getFuncMap(palette)
	tmpl, err := template.
		New("projectsList").
		Funcs(funcMap).
		Parse(projectListTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "projectList", r); err != nil {
		return "", err
	}

	return buf.String(), nil
}
