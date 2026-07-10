package views

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/arm/topo/cli/internal/catalog"
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
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r ProjectList) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
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
