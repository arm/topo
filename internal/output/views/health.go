package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
)

type HealthReport struct {
	Host       health.HostReport    `json:"host"`
	Target     *health.TargetReport `json:"target,omitempty"`
	TargetHint string               `json:"-"`
}

func PrintHealthReport(report HealthReport, w io.Writer, format term.Format, verbose bool) error {
	if format == term.JSON {
		return Print(report, w, format)
	}

	out, err := renderHealthReport(report, term.IsTTY(w), verbose)
	if err != nil {
		return fmt.Errorf("render view as plain text: %w", err)
	}
	if _, err := fmt.Fprint(w, out); err != nil {
		return fmt.Errorf("write view output: %w", err)
	}
	return nil
}

func (r HealthReport) AsPlain(isTTY bool) (string, error) {
	return renderHealthReport(r, isTTY, false)
}

func (r HealthReport) AsJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
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
{{- if showPassedSummary .Host.Dependencies }}
{{ successStatus }}All checks passed
{{- end }}
{{- range $hostCheckRow := visibleChecks .Host.Dependencies }}
{{ template "checkRow" $hostCheckRow }}
{{- end }}

{{ if .Target }}{{ targetHeading .Target.Destination -}}
  {{- $targetChecks := targetChecks .Target }}
  {{- if showPassedSummary $targetChecks }}
{{ successStatus }}All checks passed
  {{- end }}
  {{- range $targetCheckRow := visibleChecks $targetChecks }}
{{ template "checkRow" $targetCheckRow }}
  {{- end }}
{{- else -}}
{{ sectionHeading "Target" }}
{{ .TargetHint }}
{{- end }}

`

func renderHealthReport(report HealthReport, isTTY, verbose bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["status"] = healthStatusFormatter(isTTY)
	funcMap["successStatus"] = func() string {
		return healthStatusFormatter(isTTY)(health.CheckStatusOK)
	}
	funcMap["sectionHeading"] = func(heading string) string {
		return sectionHeading(heading, isTTY)
	}
	funcMap["targetHeading"] = func(destination string) string {
		return targetHeading(destination, isTTY)
	}
	funcMap["visibleChecks"] = func(checks []health.HealthCheck) []health.HealthCheck {
		return visibleHealthChecks(checks, verbose)
	}
	funcMap["showPassedSummary"] = func(checks []health.HealthCheck) bool {
		return !verbose && allHealthChecksPassed(checks)
	}
	funcMap["targetChecks"] = targetHealthChecks
	tmpl, err := template.
		New("healthcheck").
		Funcs(funcMap).
		Parse(healthReportTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, report); err != nil {
		return "", err
	}

	return buf.String(), nil
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

func visibleHealthChecks(checks []health.HealthCheck, verbose bool) []health.HealthCheck {
	if verbose {
		return checks
	}

	visible := make([]health.HealthCheck, 0, len(checks))
	for _, check := range checks {
		if check.Status != health.CheckStatusOK {
			visible = append(visible, check)
		}
	}
	return visible
}

func allHealthChecksPassed(checks []health.HealthCheck) bool {
	if len(checks) == 0 {
		return false
	}
	for _, check := range checks {
		if check.Status != health.CheckStatusOK && check.Status != health.CheckStatusInfo {
			return false
		}
	}
	return true
}

func targetHealthChecks(target *health.TargetReport) []health.HealthCheck {
	checks := make([]health.HealthCheck, 0, len(target.Dependencies)+2)
	if !target.IsLocalhost {
		checks = append(checks, target.Connectivity)
	}
	if target.IsLocalhost || target.Connectivity.Status == health.CheckStatusOK {
		checks = append(checks, target.Dependencies...)
		checks = append(checks, target.ProcessingDomainDriver)
	}
	return checks
}
