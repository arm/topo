package health

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/target"
)

// #nosec G101 -- Does not contain hardcoded credentials
const passwordAuthErrorMessage = `note: Topo does not support SSH password-based authentication. To connect, either:
- create your own SSH keys for the target, or
- run 'topo setup-keys --target %s' to let Topo generate keys and configure passwordless authentication`

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
	report.Dependencies = generateDependencyReport(targetStatus.Dependencies)

	return report
}

func GenerateReport(hostDependencies []DependencyStatus, targetStatus Status) Report {
	report := Report{}
	report.Host = generateHostReport(hostDependencies)
	report.Target = generateTargetReport(targetStatus)

	return report
}

func Check(sshTarget string, acceptNewHostKeys bool) (Report, error) {
	dependencyStatuses := CheckInstalled(HostRequiredDependencies, BinaryExistsLocally)
	opts := target.ConnectionOptions{
		AcceptNewHostKeys: acceptNewHostKeys,
		AuthProbeInput:    os.Stdin,
		AuthProbeOutput:   os.Stdout,
		Multiplex:         true,
	}
	conn := target.NewConnection(sshTarget, opts)
	targetStatus := ProbeHealthStatus(conn)
	report := GenerateReport(dependencyStatuses, targetStatus)
	if err := targetStatus.ConnectionError; err != nil {
		if errors.Is(err, target.ErrPasswordAuthentication) {
			return report, fmt.Errorf(passwordAuthErrorMessage, sshTarget)
		}
		return report, nil
	}
	return report, nil
}
