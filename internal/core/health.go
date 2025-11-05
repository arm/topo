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

func extractArmFeatures(target Target) []string {
	res := make([]string, 0)

	for _, field := range target.features {
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

type Report struct {
	HostDependencies []HealthCheck
	Connectivity     HealthCheck
	TargetFeatures   []string
}

func GenerateReport(dependencyStatuses []dependencies.Status, target Target) Report {
	report := Report{}

	availableDepsByCategory := dependencies.CollectAvailableByCategory(dependencyStatuses)

	for category, installedDependencies := range availableDepsByCategory {
		names := make([]string, len(installedDependencies))
		for i, dep := range installedDependencies {
			names[i] = dep.Dependency.Name
		}
		report.HostDependencies = append(report.HostDependencies, HealthCheck{
			Name:    category,
			Healthy: len(installedDependencies) > 0,
			Value:   strings.Join(names, ", "),
		})
	}

	report.Connectivity = HealthCheck{
		Name:    "Connected",
		Healthy: target.connectionError == nil,
		Value:   "",
	}

	report.TargetFeatures = extractArmFeatures(target)
	return report
}

const healthCheckTemplate = `
{{- define "checkRow" -}}
  {{ .Name }}:{{- if .Healthy }} ✅{{- else }} ❌{{- end }}{{- if .Value }} ({{ .Value }}){{- end }}
{{- end -}}
Host
----
{{- range $healthcheckRow := .HostDependencies }}
{{ template "checkRow" $healthcheckRow }}
{{- end }}

Target
------
{{ template "checkRow" .Connectivity }}
{{- if .Connectivity.Healthy }}
Features (Linux Host): {{ join .TargetFeatures ", " }}
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
	dependencyStatuses := dependencies.Check(dependencies.RequiredDependencies, dependencies.BinaryExistsLocally)

	target := MakeTarget(sshTarget, ExecSSH)
	report := GenerateReport(dependencyStatuses, target)
	healthCheck, err := RenderReportAsPlainText(report)
	if err != nil {
		return err
	}
	LogPrintf(healthCheck)
	return nil
}
