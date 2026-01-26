package output

import (
	"bytes"
	"encoding/json"

	"github.com/arm-debug/topo-cli/internal/catalog"
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

func PrintTemplateRepos(printer *Printer, repos []catalog.Repo) error {
	currentTemplate = getTemplate(printer, "repoTemplate", repoTemplate)
	return printer.Print(RepoCollection(repos))
}

func (r RepoCollection) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r RepoCollection) AsPlain() (string, error) {
	var buf bytes.Buffer
	if err := currentTemplate.ExecuteTemplate(&buf, "repoList", r); err != nil {
		return "", err
	}

	return buf.String(), nil
}
