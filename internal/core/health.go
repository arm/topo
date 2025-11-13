package core

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/arm-debug/topo-cli/internal/dependencies"
)

var searchFlags = map[string]string{
	"asimd": "NEON",
	"sve":   "SVE",
	"sve2":  "SVE2",
	"sme":   "SME",
	"sme2":  "SME2",
}

func ExtractArmFeatures(target Target) []string {
	res := make([]string, 0)

	for _, field := range target.Features {
		if name, ok := searchFlags[field]; ok {
			res = append(res, name)
		}
	}
	return res
}

type HealthCheck struct {
	Name    string
	Healthy bool
	Value   string
}
type HostReport struct {
	Dependencies []HealthCheck
}

type TargetReport struct {
	Connectivity HealthCheck
	Features     []string
	Dependencies []HealthCheck
}

type Report struct {
	Host   HostReport
	Target TargetReport
}

func generateDependencyReport(statuses []dependencies.Status) []HealthCheck {
	res := []HealthCheck{}
	availableDepsByCategory := dependencies.CollectAvailableByCategory(statuses)

	for category, installedDependencies := range availableDepsByCategory {
		names := make([]string, len(installedDependencies))
		for i, dep := range installedDependencies {
			names[i] = dep.Dependency.Name
		}
		res = append(res, HealthCheck{
			Name:    category,
			Healthy: len(installedDependencies) > 0,
			Value:   strings.Join(names, ", "),
		})
	}
	return res
}

func generateHostReport(statuses []dependencies.Status) HostReport {
	report := HostReport{}
	report.Dependencies = generateDependencyReport(statuses)

	return report
}

func generateTargetReport(target Target) TargetReport {
	report := TargetReport{}
	report.Connectivity = HealthCheck{
		Name:    "Connected",
		Healthy: target.ConnectionError == nil,
		Value:   "",
	}
	report.Features = ExtractArmFeatures(target)
	report.Dependencies = generateDependencyReport(target.Dependencies)

	return report
}

func GenerateReport(hostDependencies []dependencies.Status, target Target) Report {
	report := Report{}
	report.Host = generateHostReport(hostDependencies)
	report.Target = generateTargetReport(target)

	return report
}

const healthCheckTemplate = `
{{- define "checkRow" -}}
  {{ .Name }}:{{- if .Healthy }} ✅{{- else }} ❌{{- end }}{{- if .Value }} ({{ .Value }}){{- end }}
{{- end -}}
Host
----
{{- range $hostCheckRow := .Host.Dependencies }}
{{ template "checkRow" $hostCheckRow }}
{{- end }}

Target
------
{{ template "checkRow" .Target.Connectivity }}
{{- if .Target.Connectivity.Healthy }}
Features (Linux Host): {{ join .Target.Features ", " }}
{{- range $targetCheckRow := .Target.Dependencies }}
{{ template "checkRow" $targetCheckRow }}
{{- end }}
{{- end }}
`

func RenderReportAsPlainText(report Report) (string, error) {
	var buf bytes.Buffer
	funcMap := template.FuncMap{
		"join": strings.Join,
	}
	tmpl := template.Must(template.New("health").Funcs(funcMap).Parse(healthCheckTemplate))
	if err := tmpl.Execute(&buf, report); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func CheckHealth(sshTarget string) error {
	dependencyStatuses := dependencies.Check(dependencies.HostRequiredDependencies, dependencies.BinaryExistsLocally)

	target := MakeTarget(sshTarget, ExecSSH)
	report := GenerateReport(dependencyStatuses, target)
	healthCheck, err := RenderReportAsPlainText(report)
	if err != nil {
		return err
	}
	LogPrintf(healthCheck)
	return nil
}
