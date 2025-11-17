package core

import (
	"bytes"
	"fmt"
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

func ExtractArmFeatures(targetStatus TargetStatus) []string {
	res := make([]string, 0)

	for _, field := range targetStatus.Hardware.Features {
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
	Connectivity    HealthCheck
	Features        []string
	Dependencies    []HealthCheck
	SubsystemDriver HealthCheck
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

func generateTargetReport(targetStatus TargetStatus) TargetReport {
	report := TargetReport{}
	report.Connectivity = HealthCheck{
		Name:    "Connected",
		Healthy: targetStatus.ConnectionError == nil,
		Value:   "",
	}
	report.Features = ExtractArmFeatures(targetStatus)
	report.SubsystemDriver = HealthCheck{
		Name:    "Subsystem Driver (remoteproc)",
		Healthy: len(targetStatus.Hardware.RemoteCPU) > 0,
		Value:   strings.Join(targetStatus.Hardware.RemoteCPU, ", "),
	}
	report.Dependencies = generateDependencyReport(targetStatus.Dependencies)

	return report
}

func GenerateReport(hostDependencies []dependencies.Status, targetStatus TargetStatus) Report {
	report := Report{}
	report.Host = generateHostReport(hostDependencies)
	report.Target = generateTargetReport(targetStatus)

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
{{ template "checkRow" .Target.SubsystemDriver }}
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

	targetConnection := NewTargetConnection(sshTarget, ExecSSH)
	targetStatus := targetConnection.Probe()
	report := GenerateReport(dependencyStatuses, targetStatus)
	healthCheck, err := RenderReportAsPlainText(report)
	if err != nil {
		return err
	}
	fmt.Println(healthCheck)
	return nil
}
