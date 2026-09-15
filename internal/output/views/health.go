package views

import (
	"bytes"
	"text/template"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
)

type HealthReport struct {
	Host       health.HostReport
	Target     *health.TargetReport
	TargetHint string
}

func NewHealthReport(host health.HostReport, target *health.TargetReport, targetHint string) HealthReport {
	return HealthReport{Host: host, Target: target, TargetHint: targetHint}
}

type healthCheckSection struct {
	ShowPassedSummary bool
	Checks            []health.HealthCheck
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
{{- define "checkSection" -}}
{{- if .ShowPassedSummary }}
{{ successStatus }}All checks passed
{{- end }}
{{- range .Checks }}
{{ template "checkRow" . }}
{{- end -}}
{{- end -}}
{{ sectionHeading "Host" }}{{ template "checkSection" (section .Host.Dependencies) }}

{{ if .Target }}{{ sectionHeading (printf "Target: %s" .Target.Destination) }}{{ template "checkSection" (section .Target.Dependencies) }}
{{- else -}}
{{ sectionHeading "Target" }}
{{ .TargetHint }}
{{- end }}

`

type HealthReportView struct {
	HealthReport
	Verbose bool
}

func (r HealthReportView) AsPlain(isTTY bool) (string, error) {
	return renderHealthReport(r.HealthReport, isTTY, r.Verbose)
}

func (r HealthReport) AsPlain(isTTY bool) (string, error) {
	return renderHealthReport(r, isTTY, false)
}

func renderHealthReport(r HealthReport, isTTY, verbose bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["status"] = healthStatusFormatter(isTTY)
	funcMap["successStatus"] = func() string {
		return healthStatusFormatter(isTTY)(health.CheckStatusOK)
	}
	funcMap["sectionHeading"] = func(heading string) string {
		return sectionHeading(heading, isTTY)
	}
	funcMap["section"] = func(checks []health.HealthCheck) healthCheckSection {
		return newHealthCheckSection(checks, verbose)
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
	return asJSON(toJSONHealthReport(r))
}

func sectionHeading(heading string, isTTY bool) string {
	return term.Header(heading, isTTY)
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

type jsonHealthReport struct {
	Host   jsonHostReport    `json:"host"`
	Target *jsonTargetReport `json:"target,omitempty"`
}

type jsonHostReport struct {
	Dependencies []jsonHealthCheck `json:"dependencies"`
}

type jsonTargetReport struct {
	Destination            string            `json:"destination"`
	IsLocalhost            bool              `json:"isLocalhost"`
	Connectivity           jsonHealthCheck   `json:"connectivity"`
	Dependencies           []jsonHealthCheck `json:"dependencies"`
	ProcessingDomainDriver jsonHealthCheck   `json:"processingDomainDriver"`
}

type jsonHealthCheck struct {
	Name   string             `json:"name"`
	Status health.CheckStatus `json:"status"`
	Value  string             `json:"value"`
	Fix    *jsonFix           `json:"fix,omitempty"`
}

type jsonFix struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
}

func toJSONHealthReport(report HealthReport) jsonHealthReport {
	jsonReport := jsonHealthReport{
		Host: jsonHostReport{Dependencies: toJSONHealthChecks(report.Host.Dependencies)},
	}
	if report.Target != nil {
		jsonTarget := toJSONTargetReport(*report.Target)
		jsonReport.Target = &jsonTarget
	}
	return jsonReport
}

func toJSONTargetReport(report health.TargetReport) jsonTargetReport {
	jsonTarget := jsonTargetReport{
		Destination:            report.Destination,
		IsLocalhost:            report.IsLocalhost,
		Dependencies:           make([]jsonHealthCheck, 0, len(report.Dependencies)),
		ProcessingDomainDriver: jsonHealthCheck{Name: "Processing Domain Driver (remoteproc)"},
	}
	for _, check := range report.Dependencies {
		jsonCheck := toJSONHealthCheck(check)
		switch check.ID {
		case health.DependencyIDConnectivity:
			jsonTarget.Connectivity = jsonCheck
		case health.DependencyIDRemoteproc:
			jsonTarget.ProcessingDomainDriver = jsonCheck
		default:
			jsonTarget.Dependencies = append(jsonTarget.Dependencies, jsonCheck)
		}
	}
	return jsonTarget
}

func toJSONHealthChecks(checks []health.HealthCheck) []jsonHealthCheck {
	jsonChecks := make([]jsonHealthCheck, len(checks))
	for index, check := range checks {
		jsonChecks[index] = toJSONHealthCheck(check)
	}
	return jsonChecks
}

func toJSONHealthCheck(check health.HealthCheck) jsonHealthCheck {
	jsonCheck := jsonHealthCheck{Name: check.Name, Status: check.Status, Value: check.Value}
	if check.Fix != nil {
		jsonCheck.Fix = &jsonFix{Description: check.Fix.Description, Command: check.Fix.Command}
	}
	return jsonCheck
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
