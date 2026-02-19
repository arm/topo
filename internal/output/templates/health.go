package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/arm/topo/internal/health"
)

type PrintableHealthReport health.Report

const healthCheckTemplate = `
{{- define "checkRow" -}}
  {{ .Name }}:{{- if .Healthy }} ✅{{- else }} ❌{{- end }}{{- if .Value }} ({{ .Value }}){{- end }}
{{- end -}}
Host
----
{{- range $hostCheckRow := .Host.Dependencies }}
{{ template "checkRow" $hostCheckRow }}
{{- end }}

Target
------
{{- if not .Target.IsLocalhost }}
{{ template "checkRow" .Target.Connectivity }}
{{- end }}
{{- if or .Target.IsLocalhost .Target.Connectivity.Healthy }}
{{- range $targetCheckRow := .Target.Dependencies }}
{{ template "checkRow" $targetCheckRow }}
{{- end }}
{{ template "checkRow" .Target.SubsystemDriver }}
{{- end }}
`

func (r PrintableHealthReport) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	tmpl, err := template.
		New("healthcheck").
		Funcs(funcMap).
		Parse(healthCheckTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r PrintableHealthReport) AsJSON() (string, error) {
	if r.Host.Dependencies == nil {
		r.Host.Dependencies = []health.HealthCheck{}
	}
	if r.Target.Dependencies == nil {
		r.Target.Dependencies = []health.HealthCheck{}
	}

	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
}
