package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
	r := runner.NewLocal()
	commandSuccessful := func(fullCmd string) error {
		_, err := r.Run(context.Background(), fullCmd)
		return err
	}
	dependencyStatuses := PerformChecks(context.Background(), HostRequiredDependencies, r.BinaryExists, commandSuccessful)
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

func CheckTarget(ctx context.Context, dest ssh.Destination, probeOpts target.SSHAuthenticationProbeOptions) (TargetReport, error) {
	r, connErr := prepareRunner(ctx, dest, probeOpts)
	status := Status{Connection: ConnectionStatus{Destination: dest, Error: connErr}}
	if connErr == nil {
		hs := ProbeHealthStatus(ctx, r)
		status.Dependencies = hs.Dependencies
		status.Hardware = hs.Hardware
	}
	return GenerateTargetReport(status), nil
}

func prepareRunner(ctx context.Context, dest ssh.Destination, probeOpts target.SSHAuthenticationProbeOptions) (runner.Runner, error) {
	if dest.IsPlainLocalhost() {
		return runner.NewLocal(), nil
	}
	sshOpts := runner.SSHOptions{Multiplex: true}
	authProbe := target.NewSSHAuthenticationProbe(runner.NewSSH(dest, sshOpts), probeOpts)
	if err := authProbe.Probe(ctx); err != nil {
		return nil, err
	}
	return runner.NewSSH(dest, sshOpts), nil
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
	switch {
	case errors.Is(status.Error, target.ErrAuthFailed):
		check.Fix = fmt.Sprintf("run `topo setup-keys --target %s` to configure SSH keys", status.Destination)
	case errors.Is(status.Error, target.ErrHostKeyUnknown):
		check.Fix = fmt.Sprintf("run `topo health --target %s --accept-new-host-keys` to trust the target's identity", status.Destination)
	case errors.Is(status.Error, target.ErrHostKeyChanged):
		check.Fix = fmt.Sprintf("run `ssh-keygen -R %s` to remove the old host key, then retry", status.Destination.Host)
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
