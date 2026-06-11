package views

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/arm/topo/internal/catalog"
)

type TemplateList []catalog.TemplateWithCompatibility

const templateListTemplate = `
{{- define "templateRow" }}
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

{{- define "templateList" }}
{{- range . }}
{{- template "templateRow" . }}

{{ end }}
{{- end }}`

func (r TemplateList) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r TemplateList) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	tmpl, err := template.
		New("templatesList").
		Funcs(funcMap).
		Parse(templateListTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "templateList", r); err != nil {
		return "", err
	}

	return buf.String(), nil
}
