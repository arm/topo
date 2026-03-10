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

type CheckStatus string

func NewCheckStatusFromError(err error) CheckStatus {
	if err != nil {
		return CheckStatusError
	}
	return CheckStatusOK
}

const (
	CheckStatusOK      CheckStatus = "ok"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusError   CheckStatus = "error"
)

type HealthCheck struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Value  string      `json:"value"`
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

func CheckHost() HostReport {
	dependencyStatuses := CheckInstalled(HostRequiredDependencies, BinaryExistsLocally)
	return GenerateHostReport(dependencyStatuses)
}

func CheckTarget(sshTarget string, acceptNewHostKeys bool) (TargetReport, error) {
	opts := target.ConnectionOptions{
		AcceptNewHostKeys: acceptNewHostKeys,
		AuthProbeInput:    os.Stdin,
		AuthProbeOutput:   os.Stdout,
		Multiplex:         true,
		WithLoginShell:    true,
	}
	conn := target.NewConnection(sshTarget, opts)
	targetStatus := ProbeHealthStatus(conn)
	if errors.Is(targetStatus.ConnectionError, target.ErrPasswordAuthentication) {
		return TargetReport{}, fmt.Errorf(passwordAuthErrorMessage, sshTarget)
	}
	return GenerateTargetReport(targetStatus), nil
}

func GenerateHostReport(statuses []DependencyStatus) HostReport {
	report := HostReport{}
	report.Dependencies = generateDependencyReport(statuses)

	return report
}

func GenerateTargetReport(targetStatus Status) TargetReport {
	report := TargetReport{}
	report.IsLocalhost = targetStatus.SSHTarget.IsPlainLocalhost()
	report.Connectivity = HealthCheck{
		Name:   "Connected",
		Status: NewCheckStatusFromError(targetStatus.ConnectionError),
		Value:  "",
	}

	report.SubsystemDriver.Name = "Subsystem Driver (remoteproc)"
	remoteCPUs := targetStatus.Hardware.RemoteCPU
	if len(remoteCPUs) > 0 {
		names := make([]string, len(remoteCPUs))
		for i, remoteProc := range remoteCPUs {
			names[i] = remoteProc.Name
		}
		report.SubsystemDriver.Status = CheckStatusOK
		report.SubsystemDriver.Value = strings.Join(names, ", ")
	} else {
		report.SubsystemDriver.Status = CheckStatusWarning
		report.SubsystemDriver.Value = "no remoteproc devices found"
	}

	report.Dependencies = generateDependencyReport(targetStatus.Dependencies)

	return report
}

func generateDependencyReport(statuses []DependencyStatus) []HealthCheck {
	res := []HealthCheck{}
	for _, group := range groupByCategory(statuses) {
		hc := HealthCheck{Name: group.Key}

		var installedNames, errorMessages []string
		for _, dep := range group.Members {
			if dep.Error == nil {
				installedNames = append(installedNames, dep.Dependency.Name)
			} else {
				errorMessages = append(errorMessages, dep.Error.Error())
			}
		}

		if len(installedNames) > 0 {
			hc.Value = strings.Join(installedNames, ", ")
			hc.Status = CheckStatusOK
		} else {
			hc.Value = strings.Join(errorMessages, ", ")
			hc.Status = CheckStatusError
		}

		res = append(res, hc)
	}
	return res
}
