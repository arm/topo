package views

import (
	"bytes"
	"text/template"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
)

type HealthReportView struct {
	health.HealthReport
	TargetHint string
	Verbose    bool
}

type healthCheckSection struct {
	ShowPassedSummary bool
	Checks            []health.DependencyReport
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
{{ status "warning" }}{{ .TargetHint }}
{{- end }}

`

func (r HealthReportView) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["status"] = healthStatusFormatter(isTTY)
	funcMap["successStatus"] = func() string {
		return healthStatusFormatter(isTTY)(health.CheckStatusOK)
	}
	funcMap["sectionHeading"] = func(heading string) string {
		return sectionHeading(heading, isTTY)
	}
	funcMap["section"] = func(checks []health.DependencyReport) healthCheckSection {
		return newHealthCheckSection(checks, r.Verbose)
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

func (r HealthReportView) AsJSON() (string, error) {
	return asJSON(toJSONHealthReport(r.HealthReport))
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
	Dependencies []jsonDependencyReport `json:"dependencies"`
}

type jsonTargetReport struct {
	Destination            string                 `json:"destination"`
	IsLocalhost            bool                   `json:"isLocalhost"`
	Connectivity           jsonDependencyReport   `json:"connectivity"`
	Dependencies           []jsonDependencyReport `json:"dependencies"`
	ProcessingDomainDriver jsonDependencyReport   `json:"processingDomainDriver"`
}

type jsonDependencyReport struct {
	Name   string             `json:"name"`
	Status health.CheckStatus `json:"status"`
	Value  string             `json:"value"`
	Fix    *jsonFix           `json:"fix,omitempty"`
}

type jsonFix struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
}

func toJSONHealthReport(report health.HealthReport) jsonHealthReport {
	jsonReport := jsonHealthReport{
		Host: jsonHostReport{Dependencies: toJSONDependencyReports(report.Host.Dependencies)},
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
		Dependencies:           make([]jsonDependencyReport, 0, len(report.Dependencies)),
		ProcessingDomainDriver: jsonDependencyReport{Name: "Processing Domain Driver (remoteproc)"},
	}
	for _, check := range report.Dependencies {
		jsonCheck := toJSONDependencyReport(check)
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

func toJSONDependencyReports(checks []health.DependencyReport) []jsonDependencyReport {
	jsonChecks := make([]jsonDependencyReport, len(checks))
	for index, check := range checks {
		jsonChecks[index] = toJSONDependencyReport(check)
	}
	return jsonChecks
}

func toJSONDependencyReport(check health.DependencyReport) jsonDependencyReport {
	jsonCheck := jsonDependencyReport{Name: check.Name, Status: check.Status, Value: check.Value}
	if check.Fix != nil {
		jsonCheck.Fix = &jsonFix{Description: check.Fix.Description, Command: check.Fix.Command}
	}
	return jsonCheck
}

func newHealthCheckSection(checks []health.DependencyReport, verbose bool) healthCheckSection {
	section := healthCheckSection{
		Checks: make([]health.DependencyReport, 0, len(checks)),
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
