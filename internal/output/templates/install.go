package templates

import (
	"bytes"
	"encoding/json"
	"html/template"

	"github.com/arm/topo/internal/install"
)

type InstallResults []install.InstallResult

const installTemplate = `
{{- if eq (len .) 0 -}}
No binaries installed
{{- else -}}
{{- range $i, $res := . -}}
{{- if gt $i 0 }}
{{ end -}}
✓ Installed {{ $res.Binary }} to {{ $res.Location.Path }}
{{- end -}}
{{- range $path, $binaries := pathWarnings . }}

{{ $path }} is not on your PATH. To use {{ join $binaries ", " }}:
  • Add to PATH: export PATH="$PATH:{{ $path }}"
  • Or move binaries to a directory already on PATH
{{- end -}}
{{- end -}}
`

func (r InstallResults) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["pathWarnings"] = pathWarnings
	tmpl, err := template.
		New("InstallResults").
		Funcs(funcMap).
		Parse(installTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r InstallResults) AsJSON() (string, error) {
	type jsonResult struct {
		Path   string `json:"path"`
		OnPath bool   `json:"on_path"`
		Binary string `json:"binary"`
	}

	results := make([]jsonResult, len(r))
	for i, res := range r {
		results[i] = jsonResult{
			Path:   res.Location.Path,
			OnPath: res.Location.OnPath,
			Binary: res.Binary,
		}
	}

	b, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func pathWarnings(results []install.InstallResult) map[string][]string {
	out := make(map[string][]string)

	for _, res := range results {
		if res.Location.OnPath {
			continue
		}
		out[res.Location.Path] = append(out[res.Location.Path], res.Binary)
	}

	return out
}
