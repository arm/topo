package output

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"

	"github.com/arm-debug/topo-cli/internal/install"
)

// InstallResults wraps multiple install results and implements printable.
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

func getInstallTemplate() *template.Template {
	funcs := template.FuncMap{
		"join": strings.Join,
		"pathWarnings": func(results InstallResults) map[string][]string {
			warnings := make(map[string][]string)
			for _, res := range results {
				if !res.Location.OnPath {
					warnings[res.Location.Path] = append(warnings[res.Location.Path], res.Binary)
				}
			}
			return warnings
		},
	}

	return template.Must(
		template.New("installTemplate").
			Funcs(funcs).
			Parse(installTemplate),
	)
}

func PrintInstallResults(printer *Printer, results InstallResults) error {
	currentTemplate = getInstallTemplate()
	return printer.Print(results)
}

func (r InstallResults) AsPlain() (string, error) {
	var buf bytes.Buffer
	if err := currentTemplate.Execute(&buf, r); err != nil {
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
