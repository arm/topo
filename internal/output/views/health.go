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
{{ status (dependencyGroupStatus .Report.Host) }}Host
{{- range .Report.Host }}
{{ template "checkRow" . }}
{{- end }}
{{ status (dependencyGroupStatus .Report.Target) }}Target
{{- range .Report.Target }}
{{ template "checkRow" . }}
{{- end }}
{{- end -}}

{{ template "functionality" (buildFunctionalityTemplateData "Deployment" .Deployment) }}

{{ template "functionality" (buildFunctionalityTemplateData "Project management" .ProjectDiscovery) }}
`

type functionalityTemplateData struct {
	Name   string
	Report health.ReadinessReport
}

func (r HealthReport) AsPlain(isTTY bool) (string, error) {
	funcMap := getFuncMap(isTTY)
	funcMap["status"] = healthStatusFormatter(isTTY)
	funcMap["buildFunctionalityTemplateData"] = func(name string, report health.ReadinessReport) functionalityTemplateData {
		return functionalityTemplateData{Name: name, Report: report}
	}
	funcMap["functionalityHeading"] = func(name string, report health.ReadinessReport) string {
		return functionalityHeading(name, report, isTTY)
	}
	funcMap["dependencyGroupStatus"] = dependencyGroupStatus
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
	report := jsonHealthReport{
		Host: jsonHostReport{Dependencies: toJSONDependencyReports(r.Deployment.Host)},
	}
	if r.TargetDetails.Destination != "" {
		report.Target = &jsonTargetReport{
			Destination:  r.TargetDetails.Destination,
			IsLocalhost:  r.TargetDetails.IsLocalhost,
			Dependencies: toJSONDependencyReports(combinedTargetDependencies(r.Deployment.Target, r.ProjectDiscovery.Target)),
		}
	}
	return asJSON(report)
}

func combinedTargetDependencies(deployment, projectDiscovery []health.DependencyReport) []health.DependencyReport {
	targetDependencies := append([]health.DependencyReport(nil), deployment...)
	for _, dependency := range projectDiscovery {
		if dependency.ID != health.DependencyIDConnectivity {
			targetDependencies = append(targetDependencies, dependency)
		}
	}
	return targetDependencies
}

func functionalityHeading(name string, report health.ReadinessReport, isTTY bool) string {
	statusCount := countStatuses(report)
	if statusCount.errors == 0 && statusCount.warnings == 0 {
		return sectionHeading(name+": ready", isTTY)
	}

	readiness := "ready"
	if statusCount.errors > 0 {
		readiness = "not ready"
	}

	indicators := make([]string, 0, 2)
	if statusCount.errors > 0 {
		indicators = append(indicators, statusIndicator("✗", term.Red, statusCount.errors, isTTY))
	}
	if statusCount.warnings > 0 {
		indicators = append(indicators, statusIndicator("!", term.Yellow, statusCount.warnings, isTTY))
	}

	heading := fmt.Sprintf("%s: %s (%s)", name, readiness, strings.Join(indicators, " "))
	return sectionHeading(heading, isTTY)
}

func statusIndicator(symbol, color string, count uint, isTTY bool) string {
	if isTTY {
		symbol = term.Color(color, symbol)
	}
	return fmt.Sprintf("%s %d", symbol, count)
}

func countStatuses(report health.ReadinessReport) (statusCount struct{ warnings, errors uint }) {
	dependencies := append([]health.DependencyReport(nil), report.Host...)
	dependencies = append(dependencies, report.Target...)
	for _, dependency := range dependencies {
		switch dependency.Status {
		case health.CheckStatusWarning:
			statusCount.warnings++
		case health.CheckStatusError:
			statusCount.errors++
		}
	}
	return
}

func dependencyGroupStatus(dependencies []health.DependencyReport) health.CheckStatus {
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
	Destination  string                 `json:"destination"`
	IsLocalhost  bool                   `json:"isLocalhost"`
	Dependencies []jsonDependencyReport `json:"dependencies"`
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
