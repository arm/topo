package views

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
)

type HealthReportView struct {
	health.HealthReport
	Verbose bool
}

type healthCheckSection struct {
	ShowPassedSummary bool
	Checks            []health.DependencyReport
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

{{- define "checkSection" -}}
{{- if .ShowPassedSummary }}
{{ "  " }}{{ successStatus }}All checks passed
{{- end }}
{{- range .Checks }}
{{ template "checkRow" . }}
{{- end -}}
{{- end -}}

{{- define "functionality" -}}
{{ functionalityHeading .Name .StatusCounts }}
{{ status (dependencyGroupStatus .HostChecks) }}Host{{ template "checkSection" (section .HostChecks) }}
{{ status (targetStatus .TargetStatus .TargetChecks) }}{{ targetHeading }}
{{- if .TargetStatus }}
{{- if .TargetStatus.Fix }}
{{ "   " }}Fix:
{{ "     " }}{{ .TargetStatus.Fix.Description }}
{{- end }}
{{- end }}
{{- template "checkSection" (section .TargetChecks) }}
{{- end -}}

{{ template "functionality" .Deployment }}

{{ template "functionality" .ProjectDiscovery }}
`

type functionalityTemplateData struct {
	Name         string
	StatusCounts statusCounts
	HostChecks   []health.DependencyReport
	TargetChecks []health.DependencyReport
	TargetStatus *health.TargetStatus
}

type statusCounts struct {
	errors       uint
	warnings     uint
	undetermined uint
}

func buildFunctionalityTemplateData(name string, report health.ReadinessReport) functionalityTemplateData {
	data := functionalityTemplateData{
		Name:         name,
		HostChecks:   make([]health.DependencyReport, 0, len(report.Checks)),
		TargetChecks: make([]health.DependencyReport, 0, len(report.Checks)),
		TargetStatus: report.TargetStatus,
	}
	for _, check := range report.Checks {
		data.StatusCounts.addCheckStatus(check.Status)
		switch check.Scope {
		case health.DependencyScopeHost:
			data.HostChecks = append(data.HostChecks, check)
		case health.DependencyScopeTarget:
			data.TargetChecks = append(data.TargetChecks, check)
		default:
			panic("health check has an unknown scope")
		}
	}
	data.StatusCounts.addTargetStatus(report.TargetStatus)
	return data
}

func (r HealthReportView) AsPlain(palette term.Palette) (string, error) {
	funcMap := getFuncMap(palette)
	funcMap["status"] = healthStatusFormatter(palette)
	funcMap["functionalityHeading"] = func(name string, statusCounts statusCounts) string {
		return functionalityHeading(name, statusCounts, palette)
	}
	funcMap["dependencyGroupStatus"] = dependencyGroupStatus
	funcMap["targetStatus"] = targetStatus
	funcMap["dependencyValue"] = dependencyValue
	funcMap["successStatus"] = func() string {
		return healthStatusFormatter(palette)(health.CheckStatusOK)
	}
	funcMap["section"] = func(checks []health.DependencyReport) healthCheckSection {
		return newHealthCheckSection(checks, r.Verbose)
	}
	funcMap["targetHeading"] = func() string {
		if r.TargetDetails != nil && r.TargetDetails.Destination != "" {
			return "Target: " + r.TargetDetails.Destination
		}
		return "Target"
	}
	tmpl, err := template.New("functionality-healthcheck").Funcs(funcMap).Parse(functionalityHealthReportTemplate)
	if err != nil {
		return "", err
	}
	data := struct {
		Deployment       functionalityTemplateData
		ProjectDiscovery functionalityTemplateData
	}{
		Deployment:       buildFunctionalityTemplateData("Deployment", r.Deployment),
		ProjectDiscovery: buildFunctionalityTemplateData("Project management", r.ProjectDiscovery),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (r HealthReportView) AsJSON() (string, error) {
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

func functionalityHeading(name string, statusCount statusCounts, palette term.Palette) string {
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

func countStatuses(checks []health.DependencyReport, targetStatus *health.TargetStatus) statusCounts {
	counts := statusCounts{}
	for _, check := range checks {
		counts.addCheckStatus(check.Status)
	}
	counts.addTargetStatus(targetStatus)
	return counts
}

func (counts *statusCounts) addCheckStatus(status health.CheckStatus) {
	switch status {
	case health.CheckStatusError:
		counts.errors++
	case health.CheckStatusWarning:
		counts.warnings++
	case health.CheckStatusUndetermined:
		counts.undetermined++
	}
}

func (counts *statusCounts) addTargetStatus(status *health.TargetStatus) {
	if status == nil {
		return
	}
	counts.addCheckStatus(status.Status)
}

func targetStatus(status *health.TargetStatus, checks []health.DependencyReport) health.CheckStatus {
	if status != nil {
		return status.Status
	}
	return dependencyGroupStatus(checks)
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
		Checks: make([]jsonDependencyReport, 0, len(report.Checks)),
	}
	counts := countStatuses(report.Checks, report.TargetStatus)
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
	for _, check := range report.Checks {
		capability.Checks = append(capability.Checks, toJSONDependencyReport(check))
	}
	return capability
}

func toJSONDependencyReport(check health.DependencyReport) jsonDependencyReport {
	return jsonDependencyReport{
		Name:     check.Name,
		Location: dependencyLocation(check.Scope),
		Status:   check.Status,
		Value:    dependencyValue(check),
		Fix:      toJSONFix(check.Fix),
	}
}

func dependencyLocation(scope health.DependencyScope) string {
	if scope == health.DependencyScopeTarget {
		return "target"
	}
	return "host"
}

func toJSONFix(fix *health.Fix) *jsonFix {
	if fix == nil {
		return nil
	}
	return &jsonFix{Description: fix.Description, Command: fix.Command}
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
