package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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

func RenderReportAsPlainText(report Report, w io.Writer) error {
	var buf bytes.Buffer
	funcMap := template.FuncMap{
		"join": strings.Join,
	}
	tmpl := template.Must(template.New("health").Funcs(funcMap).Parse(healthCheckTemplate))
	if err := tmpl.Execute(&buf, report); err != nil {
		return err
	}

	_, err := fmt.Fprint(w, buf.String())
	return err
}

func RenderReportAsJSON(report Report, w io.Writer) error {
	if report.Host.Dependencies == nil {
		report.Host.Dependencies = []HealthCheck{}
	}
	if report.Target.Dependencies == nil {
		report.Target.Dependencies = []HealthCheck{}
	}
	if report.Target.Features == nil {
		report.Target.Features = []string{}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode report as json: %w", err)
	}
	return nil
}

func Check(sshTarget string, outputFormat output.Format, w io.Writer) error {
	report, err := CheckReport(sshTarget)
	if err != nil {
		return err
	}
	if outputFormat == output.JSONFormat {
		return RenderReportAsJSON(report, w)
	} else {
		return RenderReportAsPlainText(report, w)
	}
}

// CheckReport runs the health probes and returns the structured Report.
func CheckReport(sshTarget string) (Report, error) {
	dependencyStatuses := CheckInstalled(HostRequiredDependencies, BinaryExistsLocally)

	conn := NewConnection(sshTarget, ExecSSH)
	targetStatus := conn.Probe()
	report := GenerateReport(dependencyStatuses, targetStatus)
	return report, nil
}
