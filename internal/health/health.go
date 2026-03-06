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
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Value   string `json:"value"`
}
type HostReport struct {
	Dependencies []HealthCheck `json:"dependencies"`
}

type TargetReport struct {
	IsLocalhost     bool          `json:"isLocalhost"`
	Connectivity    HealthCheck   `json:"connectivity"`
	Dependencies    []HealthCheck `json:"dependencies"`
	SubsystemDriver HealthCheck   `json:"subsystemDriver"`
}

type Report struct {
	Host   HostReport   `json:"host"`
	Target TargetReport `json:"target"`
}

func generateDependencyReport(statuses []DependencyStatus) []HealthCheck {
	res := []HealthCheck{}
	for category, deps := range groupByCategory(statuses) {
		var installedNames, errorMessages []string
		for _, dep := range deps {
			if dep.Error == nil {
				installedNames = append(installedNames, dep.Dependency.Name)
			} else {
				errorMessages = append(errorMessages, dep.Error.Error())
			}
		}
		var value string
		if len(installedNames) > 0 {
			value = strings.Join(installedNames, ", ")
		} else {
			value = strings.Join(errorMessages, ", ")
		}
		res = append(res, HealthCheck{
			Name:    category,
			Healthy: len(installedNames) > 0,
			Value:   value,
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

	report.SubsystemDriver.Name = "Subsystem Driver (remoteproc)"
	remoteCPUs := targetStatus.Hardware.RemoteCPU
	if len(remoteCPUs) > 0 {
		names := make([]string, len(remoteCPUs))
		for i, remoteProc := range remoteCPUs {
			names[i] = remoteProc.Name
		}
		report.SubsystemDriver.Healthy = true
		report.SubsystemDriver.Value = strings.Join(names, ", ")
	} else {
		report.SubsystemDriver.Healthy = false
		report.SubsystemDriver.Value = "no remoteproc devices found"
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

func Check(sshTarget string, acceptNewHostKeys bool) (Report, error) {
	dependencyStatuses := CheckInstalled(HostRequiredDependencies, BinaryExistsLocally)
	opts := target.ConnectionOptions{
		AcceptNewHostKeys: acceptNewHostKeys,
		AuthProbeInput:    os.Stdin,
		AuthProbeOutput:   os.Stdout,
		Multiplex:         true,
		WithLoginShell:    true,
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
