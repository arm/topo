package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
)

type HealthReport struct {
	Host       health.HostReport    `json:"host"`
	Target     *health.TargetReport `json:"target,omitempty"`
	TargetHint string               `json:"-"`
	Verbose    bool                 `json:"-"`
}

type healthCheckSection struct {
	ShowPassedSummary bool
	Checks            []health.HealthCheck
}

type healthTargetSection struct {
	Destination string
	Section     healthCheckSection
}

type plainHealthReport struct {
	Host       healthCheckSection
	Target     *healthTargetSection
	TargetHint string
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
{{- if .Host.ShowPassedSummary }}
{{ successStatus }}All checks passed
{{- end }}
{{- range $hostCheckRow := .Host.Checks }}
{{ template "checkRow" $hostCheckRow }}
{{- end }}

{{ if .Target }}{{ targetHeading .Target.Destination -}}
  {{- if .Target.Section.ShowPassedSummary }}
{{ successStatus }}All checks passed
  {{- end }}
  {{- range $targetCheckRow := .Target.Section.Checks }}
{{ template "checkRow" $targetCheckRow }}
  {{- end }}
{{- else -}}
{{ sectionHeading "Target" }}
{{ .TargetHint }}
{{- end }}

`

func (r HealthReport) AsPlain(isTTY bool) (string, error) {
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
	tmpl, err := template.
		New("healthcheck").
		Funcs(funcMap).
		Parse(healthReportTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.asPlainReport()); err != nil {
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

func newHealthCheckSection(checks []health.HealthCheck, verbose bool) healthCheckSection {
	section := healthCheckSection{
		Checks: make([]health.HealthCheck, 0, len(checks)),
	}
	allPassed := len(checks) > 0

	for _, check := range checks {
		if verbose || check.Status != health.CheckStatusOK {
			section.Checks = append(section.Checks, check)
		}
		if check.Status != health.CheckStatusOK && check.Status != health.CheckStatusInfo {
			allPassed = false
		}
	}
	section.ShowPassedSummary = !verbose && allPassed

	return section
}

func (r HealthReport) asPlainReport() plainHealthReport {
	report := plainHealthReport{
		Host:       newHealthCheckSection(r.Host.Dependencies, r.Verbose),
		TargetHint: r.TargetHint,
	}
	if r.Target == nil {
		return report
	}

	targetChecks := make([]health.HealthCheck, 0, len(r.Target.Dependencies)+2)
	if !r.Target.IsLocalhost {
		targetChecks = append(targetChecks, r.Target.Connectivity)
	}
	if r.Target.IsLocalhost || r.Target.Connectivity.Status == health.CheckStatusOK {
		targetChecks = append(targetChecks, r.Target.Dependencies...)
		targetChecks = append(targetChecks, r.Target.ProcessingDomainDriver)
	}

	report.Target = &healthTargetSection{
		Destination: r.Target.Destination,
		Section:     newHealthCheckSection(targetChecks, r.Verbose),
	}
	return report
}
