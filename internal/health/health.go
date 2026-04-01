package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
)

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
	CheckStatusInfo    CheckStatus = "info"
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
	dependencyStatuses := PerformChecks(HostRequiredDependencies, BinaryExistsLocally, CommandSuccessfulLocally)
	return GenerateHostReport(dependencyStatuses)
}

type ConnectionStatus struct {
	Destination ssh.Destination
	Error       error
}

func (c ConnectionStatus) IsPlainLocalhost() bool {
	return c.Destination.IsPlainLocalhost()
}

type Status struct {
	Connection   ConnectionStatus
	Dependencies []DependencyStatus
	Hardware     HardwareProfile
}

func CheckTarget(dest ssh.Destination, probeOpts target.SSHAuthenticationProbeOptions, connectTimeout time.Duration) (TargetReport, error) {
	r, connErr := prepareRunner(dest, probeOpts, connectTimeout)
	status := Status{Connection: ConnectionStatus{Destination: dest, Error: connErr}}
	if connErr == nil {
		hs := ProbeHealthStatus(r)
		status.Dependencies = hs.Dependencies
		status.Hardware = hs.Hardware
	}
	return GenerateTargetReport(status), nil
}

func prepareRunner(dest ssh.Destination, probeOpts target.SSHAuthenticationProbeOptions, connectTimeout time.Duration) (commandRunner, error) {
	if dest.IsPlainLocalhost() {
		local := runner.NewLocal()
		return &local, nil
	}
	sshRunner := runner.NewSSH(dest, runner.SSHOptions{Multiplex: true, ConnectTimeout: connectTimeout})
	authProbe := target.NewSSHAuthenticationProbe(&sshRunner, probeOpts)
	if err := authProbe.Probe(); err != nil {
		return nil, err
	}
	return &sshRunner, nil
}

func GenerateHostReport(statuses []DependencyStatus) HostReport {
	report := HostReport{}
	report.Dependencies = generateDependencyReport(statuses)

	return report
}

func GenerateTargetReport(targetStatus Status) TargetReport {
	report := TargetReport{}
	report.IsLocalhost = targetStatus.Connection.IsPlainLocalhost()
	report.Connectivity = connectivityCheck(targetStatus.Connection)

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
		report.SubsystemDriver.Status = CheckStatusInfo
		report.SubsystemDriver.Value = "no remoteproc devices found"
	}

	report.Dependencies = generateDependencyReport(targetStatus.Dependencies)

	return report
}

func connectivityCheck(status ConnectionStatus) HealthCheck {
	check := HealthCheck{
		Name:   "Connectivity",
		Status: NewCheckStatusFromError(status.Error),
	}
	if status.Error == nil {
		return check
	}

	check.Value = status.Error.Error()
	if errors.Is(status.Error, target.ErrPasswordAuthentication) {
		check.Fix = fmt.Sprintf("run `topo setup-keys --target %s` or manually setup SSH keys for the target", status.Destination)
	}
	return check
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
