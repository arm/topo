package views

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
)

type HealthReport struct {
	TargetDetails    health.TargetDetails
	Deployment       health.ReadinessReport
	ProjectDiscovery health.ReadinessReport
}

const functionalityHealthReportTemplate = `
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
{{- define "indentedCheckRow" -}}
{{ "  " }}{{ status .Status }}{{ .Name }}{{- if .Value }} ({{ .Value }}){{- end }}
{{- if .Fix }}
    Fix:
      {{ .Fix.Description }}
  {{- if .Fix.Command }}
    Command:
      {{ .Fix.Command }}
  {{- end }}
{{- end -}}
{{- end -}}
{{- define "functionality" -}}
{{ functionalityHeading .Name .Report }}
{{ status (dependencyStatus .Report.Host) }}Host
{{- range .Report.Host }}
{{ template "indentedCheckRow" . }}
{{- end }}
{{ status (dependencyStatus .Report.Target) }}Target
{{- range .Report.Target }}
{{ template "indentedCheckRow" . }}
{{- end }}
{{- end -}}
{{ template "functionality" (functionality "Deployment" .Deployment) }}

{{ template "functionality" (functionality "Project management" .ProjectDiscovery) }}
`

type functionalityTemplateData struct {
	Name   string
	Report health.ReadinessReport
}

func (r HealthReport) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["status"] = healthStatusFormatter(isTTY)
	funcMap["functionality"] = func(name string, report health.ReadinessReport) functionalityTemplateData {
		return functionalityTemplateData{Name: name, Report: report}
	}
	funcMap["functionalityHeading"] = func(name string, report health.ReadinessReport) string {
		return functionalityHeading(name, report, isTTY)
	}
	funcMap["dependencyStatus"] = dependencyStatus
	tmpl, err := template.New("functionality-healthcheck").Funcs(funcMap).Parse(functionalityHealthReportTemplate)
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
	targetDependencies := legacyTargetDependencies(r.Deployment.Target, r.ProjectDiscovery.Target)
	return asJSON(toJSONHealthReport(legacyHealthReport{
		HostDependencies:   r.Deployment.Host,
		TargetDependencies: targetDependencies,
		TargetDetails:      r.TargetDetails,
	}))
}

type legacyHealthReport struct {
	HostDependencies   []health.DependencyReport
	TargetDependencies []health.DependencyReport
	TargetDetails      health.TargetDetails
}

func legacyTargetDependencies(deployment, projectDiscovery []health.DependencyReport) []health.DependencyReport {
	targetDependencies := append([]health.DependencyReport(nil), deployment...)
	for _, dependency := range projectDiscovery {
		if dependency.ID != health.DependencyIDConnectivity {
			targetDependencies = append(targetDependencies, dependency)
		}
	}
	return targetDependencies
}

func functionalityHeading(name string, report health.ReadinessReport, isTTY bool) string {
	status, unreadyDependencies := functionalityStatus(report)
	readiness := "ready"
	indicator := ""
	color := term.Green
	if unreadyDependencies > 0 {
		readiness = "not ready"
		icon := "✗"
		color = term.Red
		if status == health.CheckStatusWarning {
			icon = "!"
			color = term.Yellow
			readiness = "ready"
		}
		indicator = fmt.Sprintf("%s %d", icon, unreadyDependencies)
		readiness += " (" + indicator + ")"
	}
	heading := sectionHeading(name+": "+readiness, isTTY)
	if !isTTY || indicator == "" {
		return heading
	}
	return strings.Replace(heading, indicator, term.Color(color, indicator), 1)
}

func functionalityStatus(report health.ReadinessReport) (health.CheckStatus, int) {
	dependencies := append([]health.DependencyReport(nil), report.Host...)
	dependencies = append(dependencies, report.Target...)
	unreadyDependencies := 0
	for _, dependency := range dependencies {
		if dependency.Status == health.CheckStatusWarning || dependency.Status == health.CheckStatusError {
			unreadyDependencies++
		}
	}
	return dependencyStatus(dependencies), unreadyDependencies
}

func dependencyStatus(dependencies []health.DependencyReport) health.CheckStatus {
	status := health.CheckStatusOK
	for _, dependency := range dependencies {
		if dependency.Status == health.CheckStatusError {
			return health.CheckStatusError
		}
		if dependency.Status == health.CheckStatusWarning {
			status = health.CheckStatusWarning
		}
	}
	return status
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

func toJSONHealthReport(report legacyHealthReport) jsonHealthReport {
	jsonReport := jsonHealthReport{
		Host: jsonHostReport{Dependencies: toJSONDependencyReports(report.HostDependencies)},
	}
	if report.TargetDetails.Destination != "" {
		jsonTarget := toJSONTargetReport(report.TargetDependencies, report.TargetDetails)
		jsonReport.Target = &jsonTarget
	}
	return jsonReport
}

func toJSONTargetReport(dependencies []health.DependencyReport, details health.TargetDetails) jsonTargetReport {
	jsonTarget := jsonTargetReport{
		Destination:            details.Destination,
		IsLocalhost:            details.IsLocalhost,
		Dependencies:           make([]jsonDependencyReport, 0, len(dependencies)),
		ProcessingDomainDriver: jsonDependencyReport{Name: "Processing Domain Driver (remoteproc)"},
	}
	for _, check := range dependencies {
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
