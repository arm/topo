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
	TargetDetails    *health.TargetDetails
	Deployment       health.ReadinessReport
	ProjectDiscovery health.ReadinessReport
}

const functionalityHealthReportTemplate = `
{{- define "checkRow" -}}
{{ "  " }}{{ status .Status }}{{ .Name }}{{- if dependencyValue . }} ({{ dependencyValue . }}){{- end }}
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
{{ status (targetStatus .Report) }}Target
{{- if .Report.TargetStatus }}
{{- if .Report.TargetStatus.Fix }}
{{ "   " }}Fix:
{{ "     " }}{{ .Report.TargetStatus.Fix.Description }}
{{- end }}
{{- end }}
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

func (r HealthReport) AsPlain(palette term.Palette) (string, error) {
	funcMap := getFuncMap(palette)
	funcMap["status"] = healthStatusFormatter(palette)
	funcMap["buildFunctionalityTemplateData"] = func(name string, report health.ReadinessReport) functionalityTemplateData {
		return functionalityTemplateData{Name: name, Report: report}
	}
	funcMap["functionalityHeading"] = func(name string, report health.ReadinessReport) string {
		return functionalityHeading(name, report, palette)
	}
	funcMap["dependencyGroupStatus"] = dependencyGroupStatus
	funcMap["targetStatus"] = targetStatus
	funcMap["dependencyValue"] = dependencyValue
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
	report := jsonHealthReport{Capabilities: make([]jsonCapabilityReport, 0, 2)}
	for _, capability := range []jsonCapabilityReport{
		toJSONCapabilityReport("Deployment", r.Deployment),
		toJSONCapabilityReport("Project management", r.ProjectDiscovery),
	} {
		if len(capability.Checks) == 0 && capability.Status == health.CheckStatusOK && capability.Fix == nil {
			continue
		}
		report.Capabilities = append(report.Capabilities, capability)
	}
	return asJSON(report)
}

func functionalityHeading(name string, report health.ReadinessReport, palette term.Palette) string {
	statusCount := countStatuses(report)
	if statusCount.errors == 0 && statusCount.undetermined == 0 && statusCount.warnings == 0 {
		return sectionHeading(name+": ready", palette)
	}

	readiness := "ready"
	if statusCount.errors > 0 {
		readiness = "not ready"
	} else if statusCount.undetermined > 0 {
		readiness = "undetermined"
	}

	indicators := make([]string, 0, 3)
	if statusCount.errors > 0 {
		indicators = append(indicators, statusIndicator("✗", term.Red, statusCount.errors, palette))
	}
	if statusCount.warnings > 0 {
		indicators = append(indicators, statusIndicator("!", term.Yellow, statusCount.warnings, palette))
	}
	if statusCount.undetermined > 0 {
		indicators = append(indicators, statusIndicator("?", term.Gray, statusCount.undetermined, palette))
	}

	heading := fmt.Sprintf("%s: %s (%s)", name, readiness, strings.Join(indicators, " "))
	return sectionHeading(heading, palette)
}

func statusIndicator(symbol, color string, count uint, palette term.Palette) string {
	return fmt.Sprintf("%s %d", palette.Color(color, symbol), count)
}

func countStatuses(report health.ReadinessReport) (statusCount struct{ errors, warnings, undetermined uint }) {
	dependencies := append([]health.DependencyReport(nil), report.Host...)
	dependencies = append(dependencies, report.Target...)
	for _, dependency := range dependencies {
		switch dependency.Status {
		case health.CheckStatusError:
			statusCount.errors++
		case health.CheckStatusWarning:
			statusCount.warnings++
		case health.CheckStatusUndetermined:
			statusCount.undetermined++
		}
	}
	if report.TargetStatus == nil {
		return
	}
	switch report.TargetStatus.Status {
	case health.CheckStatusWarning:
		statusCount.warnings++
	case health.CheckStatusError:
		statusCount.errors++
	}
	return
}

func targetStatus(report health.ReadinessReport) health.CheckStatus {
	if report.TargetStatus != nil {
		return report.TargetStatus.Status
	}
	return dependencyGroupStatus(report.Target)
}

func dependencyValue(report health.DependencyReport) string {
	if report.Status != health.CheckStatusUndetermined {
		return report.Value
	}
	return "not checked: requires " + formatBlockers(report.BlockedBy)
}

func formatBlockers(blockers []health.DependencyBlocker) string {
	references := make([]string, len(blockers))
	for i, blocker := range blockers {
		references[i] = dependencyScopePossessive(blocker.Scope) + " " + blocker.Name
	}

	switch len(references) {
	case 0:
		return ""
	case 1:
		return references[0]
	case 2:
		return strings.Join(references, " and ")
	default:
		return strings.Join(references[:len(references)-1], ", ") + ", and " + references[len(references)-1]
	}
}

func dependencyScopePossessive(scope health.DependencyScope) string {
	if scope == health.DependencyScopeTarget {
		return "target's"
	}
	return "host's"
}

func dependencyGroupStatus(dependencies []health.DependencyReport) health.CheckStatus {
	status := health.CheckStatusOK
	for _, dependency := range dependencies {
		if dependency.Status == health.CheckStatusError {
			return health.CheckStatusError
		}
		if dependency.Status == health.CheckStatusUndetermined && status != health.CheckStatusError {
			status = health.CheckStatusUndetermined
		}
		if dependency.Status == health.CheckStatusWarning && status == health.CheckStatusOK {
			status = health.CheckStatusWarning
		}
	}
	return status
}

func sectionHeading(heading string, palette term.Palette) string {
	return term.Header(heading, palette)
}

func healthStatusFormatter(palette term.Palette) func(health.CheckStatus) string {
	return func(status health.CheckStatus) string {
		label, color := " ✗ ", term.Red
		switch status {
		case health.CheckStatusOK:
			label, color = " ✓ ", term.Green
		case health.CheckStatusWarning:
			label, color = " ! ", term.Yellow
		case health.CheckStatusInfo:
			label, color = " i ", term.Blue
		case health.CheckStatusUndetermined:
			label, color = " ? ", term.Gray
		}
		return palette.Color(color, label)
	}
}

type jsonHealthReport struct {
	Capabilities []jsonCapabilityReport `json:"capabilities"`
}

type jsonCapabilityReport struct {
	Name   string                 `json:"name"`
	Status health.CheckStatus     `json:"status"`
	Fix    *jsonFix               `json:"fix,omitempty"`
	Checks []jsonDependencyReport `json:"checks"`
}

type jsonDependencyReport struct {
	Name     string             `json:"name"`
	Location string             `json:"location"`
	Status   health.CheckStatus `json:"status"`
	Value    string             `json:"value"`
	Fix      *jsonFix           `json:"fix,omitempty"`
}

type jsonFix struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
}

func toJSONCapabilityReport(name string, report health.ReadinessReport) jsonCapabilityReport {
	capability := jsonCapabilityReport{
		Name:   name,
		Status: health.CheckStatusOK,
		Checks: make([]jsonDependencyReport, 0, len(report.Host)+len(report.Target)),
	}
	counts := countStatuses(report)
	switch {
	case counts.errors > 0:
		capability.Status = health.CheckStatusError
	case counts.undetermined > 0:
		capability.Status = health.CheckStatusUndetermined
	case counts.warnings > 0:
		capability.Status = health.CheckStatusWarning
	}
	if report.TargetStatus != nil {
		capability.Fix = toJSONFix(report.TargetStatus.Fix)
	}
	for _, check := range report.Host {
		capability.Checks = append(capability.Checks, toJSONDependencyReport(check, "host"))
	}
	for _, check := range report.Target {
		capability.Checks = append(capability.Checks, toJSONDependencyReport(check, "target"))
	}
	return capability
}

func toJSONDependencyReport(check health.DependencyReport, location string) jsonDependencyReport {
	return jsonDependencyReport{
		Name:     check.Name,
		Location: location,
		Status:   check.Status,
		Value:    dependencyValue(check),
		Fix:      toJSONFix(check.Fix),
	}
}

func toJSONFix(fix *health.Fix) *jsonFix {
	if fix == nil {
		return nil
	}
	return &jsonFix{Description: fix.Description, Command: fix.Command}
}
