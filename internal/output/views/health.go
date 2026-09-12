package views

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
)

type HealthReport struct {
	Host       hostReport    `json:"host"`
	Target     *targetReport `json:"target,omitempty"`
	TargetHint string        `json:"-"`
}

func NewHealthReport(host health.HostReport, target *health.TargetReport, targetHint string) HealthReport {
	report := HealthReport{
		Host:       toViewHostReport(host),
		TargetHint: targetHint,
	}
	if target != nil {
		viewTarget := toViewTargetReport(*target)
		report.Target = &viewTarget
	}
	return report
}

const healthReportTemplate = `
{{- define "checkRow" -}}
{{ status .Status }}{{ .Name }}{{- if .Value }} ({{ .Value }}){{- end }}
{{- if .Fix }}
   Fix:
     {{ .Fix.Description }}
  {{- if .Fix.Command }}
   Command:
     {{ .Fix.Command }}
  {{- end }}
{{- end -}}
{{- end -}}
{{ sectionHeading "Host" }}
{{- range $hostCheckRow := .Host.Dependencies }}
{{ template "checkRow" $hostCheckRow }}
{{- end }}

{{ if .Target }}{{ targetHeading .Target.Destination -}}
  {{- if not .Target.IsLocalhost }}
{{ template "checkRow" .Target.Connectivity }}
  {{- end }}
  {{- if or .Target.IsLocalhost (isOK .Target.Connectivity.Status) }}
    {{- range $targetCheckRow := .Target.Dependencies }}
{{ template "checkRow" $targetCheckRow }}
    {{- end }}
  {{- end }}
{{- else -}}
{{ sectionHeading "Target" }}
{{ .TargetHint }}
{{- end }}

`

func (r HealthReport) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["status"] = healthStatusFormatter(isTTY)
	funcMap["sectionHeading"] = func(heading string) string {
		return sectionHeading(heading, isTTY)
	}
	funcMap["targetHeading"] = func(destination string) string {
		return targetHeading(destination, isTTY)
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
	return asJSON(r)
}

func sectionHeading(heading string, isTTY bool) string {
	return term.Header(heading, isTTY)
}

func targetHeading(destination string, isTTY bool) string {
	return sectionHeading(fmt.Sprintf("Target: %s", destination), isTTY)
}

func healthStatusFormatter(isTTY bool) func(health.CheckStatus) string {
	return func(status health.CheckStatus) string {
		label, color := " ✗ ", term.Red
		switch status {
		case health.CheckStatusOK:
			label, color = " ✓ ", term.Green
		case health.CheckStatusWarning:
			label, color = " ! ", term.Yellow
		case health.CheckStatusInfo:
			label, color = " i ", term.Blue
		}
		if !isTTY {
			return label
		}
		return term.Color(color, label)
	}
}

type hostReport struct {
	Dependencies []healthCheck `json:"dependencies"`
}

type targetReport struct {
	Destination  string        `json:"destination"`
	IsLocalhost  bool          `json:"isLocalhost"`
	Connectivity healthCheck   `json:"connectivity"`
	Dependencies []healthCheck `json:"dependencies"`
}

type healthCheck struct {
	Name   string             `json:"name"`
	Status health.CheckStatus `json:"status"`
	Value  string             `json:"value"`
	Fix    *fix               `json:"fix,omitempty"`
}

type fix struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
}

func toViewHostReport(report health.HostReport) hostReport {
	return hostReport{Dependencies: toViewHealthCheckList(report.Dependencies)}
}

func toViewTargetReport(report health.TargetReport) targetReport {
	return targetReport{
		Destination:  report.Destination,
		IsLocalhost:  report.IsLocalhost,
		Connectivity: toViewHealthCheck(report.Connectivity),
		Dependencies: toViewHealthCheckList(report.Dependencies),
	}
}

func toViewHealthCheckList(checks []health.HealthCheck) []healthCheck {
	viewChecks := make([]healthCheck, len(checks))
	for index, check := range checks {
		viewChecks[index] = toViewHealthCheck(check)
	}
	return viewChecks
}

func toViewHealthCheck(check health.HealthCheck) healthCheck {
	viewCheck := healthCheck{Name: check.Name, Status: check.Status, Value: check.Value}
	if check.Fix != nil {
		viewCheck.Fix = &fix{Description: check.Fix.Description, Command: check.Fix.Command}
	}
	return viewCheck
}
