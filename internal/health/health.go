package health

import (
	"encoding/json"
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
	Fix    string      `json:"fix,omitempty"`
}

type HostReport struct {
	Dependencies []HealthCheck `json:"dependencies"`
}

func (r HostReport) MarshalJSON() ([]byte, error) {
	type Alias HostReport
	if r.Dependencies == nil {
		r.Dependencies = []HealthCheck{}
	}
	return json.Marshal(Alias(r))
}

type TargetReport struct {
	IsLocalhost     bool          `json:"isLocalhost"`
	Connectivity    HealthCheck   `json:"connectivity"`
	Dependencies    []HealthCheck `json:"dependencies"`
	SubsystemDriver HealthCheck   `json:"subsystemDriver"`
}

func (r TargetReport) MarshalJSON() ([]byte, error) {
	type Alias TargetReport
	if r.Dependencies == nil {
		r.Dependencies = []HealthCheck{}
	}
	return json.Marshal(Alias(r))
}

func CheckHost() HostReport {
	dependencyStatuses := PerformChecks(HostRequiredDependencies, BinaryExistsLocally)
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
	for _, ds := range statuses {
		hc := HealthCheck{Name: ds.Dependency.Label}
		if ds.Error == nil {
			hc.Status = CheckStatusOK
			hc.Value = ds.Dependency.Binary
		} else {
			if _, ok := errors.AsType[WarningError](ds.Error); ok {
				hc.Status = CheckStatusWarning
			} else {
				hc.Status = CheckStatusError
			}
			hc.Value = ds.Error.Error()
			hc.Fix = ds.Fix
		}
		res = append(res, hc)
	}
	return res
}
