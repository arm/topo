package health

import (
	"strings"

	"github.com/arm-debug/topo-cli/internal/ssh"
)

var searchFlags = map[string]string{
	"asimd": "NEON",
	"sve":   "SVE",
	"sve2":  "SVE2",
	"sme":   "SME",
	"sme2":  "SME2",
}

func ExtractArmFeatures(targetStatus Status) []string {
	if len(targetStatus.Hardware.HostProcessor) == 0 {
		return nil
	}

	var res []string
	for _, field := range targetStatus.Hardware.HostProcessor[0].Features {
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
	IsLocalhost     bool
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
	report.IsLocalhost = targetStatus.SSHTarget.IsPlainLocalhost()
	report.Connectivity = HealthCheck{
		Name:    "Connected",
		Healthy: targetStatus.ConnectionError == nil,
		Value:   "",
	}
	report.SubsystemDriver = HealthCheck{
		Name:    "Subsystem Driver (remoteproc)",
		Healthy: len(targetStatus.Hardware.RemoteCPU) > 0,
	}
	var remoteProcNames []string
	for _, remoteProc := range targetStatus.Hardware.RemoteCPU {
		remoteProcNames = append(remoteProcNames, remoteProc.Name)
	}
	report.SubsystemDriver.Value = strings.Join(remoteProcNames, ", ")
	report.Features = ExtractArmFeatures(targetStatus)
	report.Dependencies = generateDependencyReport(targetStatus.Dependencies)

	return report
}

func GenerateReport(hostDependencies []DependencyStatus, targetStatus Status) Report {
	report := Report{}
	report.Host = generateHostReport(hostDependencies)
	report.Target = generateTargetReport(targetStatus)

	return report
}

func Check(sshTarget string) (Report, error) {
	dependencyStatuses := CheckInstalled(HostRequiredDependencies, BinaryExistsLocally)

	conn := NewConnection(sshTarget, ssh.ExecSSH)
	targetStatus := conn.Probe()
	report := GenerateReport(dependencyStatuses, targetStatus)
	return report, nil
}
