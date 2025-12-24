package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/arm-debug/topo-cli/internal/output"
)

var searchFlags = map[string]string{
	"asimd": "NEON",
	"sve":   "SVE",
	"sve2":  "SVE2",
	"sme":   "SME",
	"sme2":  "SME2",
}

func ExtractArmFeatures(targetStatus Status) []string {
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

func generateDependencyReport(statuses []DependencyStatus) []HealthCheck {
	res := []HealthCheck{}
	availableDepsByCategory := CollectAvailableByCategory(statuses)

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

func generateHostReport(statuses []DependencyStatus) HostReport {
	report := HostReport{}
	report.Dependencies = generateDependencyReport(statuses)

	return report
}

func generateTargetReport(targetStatus Status) TargetReport {
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

func GenerateReport(hostDependencies []DependencyStatus, targetStatus Status) Report {
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

func Check(sshTarget string, printer *output.Printer) error {
	report, err := CheckReport(sshTarget)
	if err != nil {
		return err
	}
	return printer.Print(report)
}

// CheckReport runs the health probes and returns the structured Report.
func CheckReport(sshTarget string) (Report, error) {
	dependencyStatuses := CheckInstalled(HostRequiredDependencies, BinaryExistsLocally)

	conn := NewConnection(sshTarget, ExecSSH)
	targetStatus := conn.Probe()
	report := GenerateReport(dependencyStatuses, targetStatus)
	return report, nil
}

func (r Report) AsPlain() (string, error) {
	var buf bytes.Buffer
	funcMap := template.FuncMap{
		"join": strings.Join,
	}
	tmpl := template.Must(template.New("health").Funcs(funcMap).Parse(healthCheckTemplate))
	if err := tmpl.Execute(&buf, r); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r Report) AsJSON() (string, error) {
	if r.Host.Dependencies == nil {
		r.Host.Dependencies = []HealthCheck{}
	}
	if r.Target.Dependencies == nil {
		r.Target.Dependencies = []HealthCheck{}
	}
	if r.Target.Features == nil {
		r.Target.Features = []string{}
	}

	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report as json: %w", err)
	}
	return string(b), nil
}
