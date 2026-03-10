package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/arm/topo/internal/health"
)

type PrintableHealthReport struct {
	Host       health.HostReport    `json:"host"`
	Target     *health.TargetReport `json:"target,omitempty"`
	TargetHint string               `json:"-"`
}

const healthCheckTemplate = `
{{- define "checkRow" -}}
  {{ .Name }}:{{ statusIcon .Status }}{{- if .Value }} ({{ .Value }}){{- end }}
{{- end -}}
Host
----
{{- range $hostCheckRow := .Host.Dependencies }}
{{ template "checkRow" $hostCheckRow }}
{{- end }}

Target
------
{{- if .Target }}
  {{- if not .Target.IsLocalhost }}
{{ template "checkRow" .Target.Connectivity }}
  {{- end }}
  {{- if or .Target.IsLocalhost (isOK .Target.Connectivity.Status) }}
    {{- range $targetCheckRow := .Target.Dependencies }}
{{ template "checkRow" $targetCheckRow }}
    {{- end }}
{{ template "checkRow" .Target.SubsystemDriver }}
  {{- end }}
{{- else }}
ℹ️ {{ .TargetHint }}
{{- end }}
`

func (r PrintableHealthReport) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["statusIcon"] = func(s health.CheckStatus) string {
		switch s {
		case health.CheckStatusOK:
			return " ✅"
		case health.CheckStatusWarning:
			return " ⚠️"
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
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
}
