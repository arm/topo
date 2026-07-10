package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/arm/topo/cli/internal/health"
)

type HealthReport struct {
	Host       health.HostReport    `json:"host"`
	Target     *health.TargetReport `json:"target,omitempty"`
	TargetHint string               `json:"-"`
}

const healthReportTemplate = `
{{- define "checkRow" -}}
{{ .Name }}:{{ statusIcon .Status }}{{- if .Value }} ({{ .Value }}){{- end }}
{{- if .Fix }}
  Fix:
    {{ .Fix.Description }}
  {{- if .Fix.Command }}
  Command:
    {{ .Fix.Command }}
  {{- end }}
{{- end -}}
{{- end -}}
Host
----
{{- range $hostCheckRow := .Host.Dependencies }}
{{ template "checkRow" $hostCheckRow }}
{{- end }}

Target
------
{{- if .Target }}
Destination: {{ .Target.Destination }}
  {{- if not .Target.IsLocalhost }}
{{ template "checkRow" .Target.Connectivity }}
  {{- end }}
  {{- if or .Target.IsLocalhost (isOK .Target.Connectivity.Status) }}
    {{- range $targetCheckRow := .Target.Dependencies }}
{{ template "checkRow" $targetCheckRow }}
    {{- end }}
{{ template "checkRow" .Target.ProcessingDomainDriver }}
  {{- end }}
{{- else }}
ℹ️ {{ .TargetHint }}
{{- end }}
`

func (r HealthReport) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["statusIcon"] = func(s health.CheckStatus) string {
		switch s {
		case health.CheckStatusOK:
			return " ✅"
		case health.CheckStatusWarning:
			return " ⚠️"
		case health.CheckStatusInfo:
			return " ℹ️"
		default:
			return " ❌"
		}
	}
	funcMap["isOK"] = func(s health.CheckStatus) bool {
		return s == health.CheckStatusOK
	}
	tmpl, err := template.
		New("healthcheck").
		Funcs(funcMap).
		Parse(healthReportTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r HealthReport) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
}
