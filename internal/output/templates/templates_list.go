package templates

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/arm/topo/internal/catalog"
)

type RepoCollection []catalog.Repo

const repoTemplate = `
{{- define "featuresRow" -}}
{{- if .Features }}
  Features: {{ join .Features ", " }}
{{- end }}
{{- end }}

{{- define "descriptionRow" -}}
{{- if .Description }}
{{ wrap .Description }}
{{- end }}
{{- end }}

{{- define "repoRow" }}
{{- cyan .Id }} | {{ blue .Url }} | {{ yellow .Ref }}
{{- template "featuresRow" . }}
{{- template "descriptionRow" . }}
{{- end }}

{{- define "repoList" }}
{{- range . }}
{{- template "repoRow" . }}

{{ end }}
{{- end }}`

func (r RepoCollection) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r RepoCollection) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	tmpl, err := template.
		New("templatesList").
		Funcs(funcMap).
		Parse(repoTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "repoList", r); err != nil {
		return "", err
	}

	return buf.String(), nil
}
