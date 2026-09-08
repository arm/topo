package views

import (
	"bytes"
	"fmt"
	"io"
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

type healthCheckSection struct {
	ShowPassedSummary bool
	Checks            []healthCheck
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

func renderHealthReport(r HealthReport, isTTY, verbose bool) (string, error) {
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
	if err := tmpl.Execute(&buf, r.asPlainReport(verbose)); err != nil {
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
	Destination            string        `json:"destination"`
	IsLocalhost            bool          `json:"isLocalhost"`
	Connectivity           healthCheck   `json:"connectivity"`
	Dependencies           []healthCheck `json:"dependencies"`
	ProcessingDomainDriver healthCheck   `json:"processingDomainDriver"`
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
	target := targetReport{
		Destination:            report.Destination,
		IsLocalhost:            report.IsLocalhost,
		Connectivity:           toViewHealthCheck(report.Connectivity),
		Dependencies:           make([]healthCheck, 0, len(report.Dependencies)),
		ProcessingDomainDriver: healthCheck{Name: "Processing Domain Driver (remoteproc)"},
	}
	for _, check := range report.Dependencies {
		viewCheck := toViewHealthCheck(check)
		if check.ID == health.DependencyIDRemoteproc {
			target.ProcessingDomainDriver = viewCheck
			continue
		}
		target.Dependencies = append(target.Dependencies, viewCheck)
	}
	return target
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

func newHealthCheckSection(checks []healthCheck, verbose bool) healthCheckSection {
	section := healthCheckSection{
		Checks: make([]healthCheck, 0, len(checks)),
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

func (r HealthReport) asPlainReport(verbose bool) plainHealthReport {
	report := plainHealthReport{
		Host:       newHealthCheckSection(r.Host.Dependencies, verbose),
		TargetHint: r.TargetHint,
	}
	if r.Target == nil {
		return report
	}

	targetChecks := make([]healthCheck, 0, len(r.Target.Dependencies)+2)
	if !r.Target.IsLocalhost {
		targetChecks = append(targetChecks, r.Target.Connectivity)
	}
	if r.Target.IsLocalhost || r.Target.Connectivity.Status == health.CheckStatusOK {
		targetChecks = append(targetChecks, r.Target.Dependencies...)
		targetChecks = append(targetChecks, r.Target.ProcessingDomainDriver)
	}

	report.Target = &healthTargetSection{
		Destination: r.Target.Destination,
		Section:     newHealthCheckSection(targetChecks, verbose),
	}
	return report
}
