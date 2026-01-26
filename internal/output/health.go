package output

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/arm-debug/topo-cli/internal/health"
)

type Report health.Report

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
Features (Linux Host): {{ join .Target.Features ", " }}
{{- range $targetCheckRow := .Target.Dependencies }}
{{ template "checkRow" $targetCheckRow }}
{{- end }}
{{ template "checkRow" .Target.SubsystemDriver }}
{{- end }}
`

func PrintHealthReport(printer *Printer, report health.Report) error {
	currentTemplate = getTemplate(printer, "healthTemplate", healthCheckTemplate)
	return printer.Print(Report(report))
}

func (r Report) AsPlain() (string, error) {
	var buf bytes.Buffer
	if err := currentTemplate.Execute(&buf, r); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r Report) AsJSON() (string, error) {
	if r.Host.Dependencies == nil {
		r.Host.Dependencies = []health.HealthCheck{}
	}
	if r.Target.Dependencies == nil {
		r.Target.Dependencies = []health.HealthCheck{}
	}
	if r.Target.Features == nil {
		r.Target.Features = []string{}
	}

	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
}
