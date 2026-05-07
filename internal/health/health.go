package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
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
	Name       string      `json:"name"`
	Status     CheckStatus `json:"status"`
	Value      string      `json:"value"`
	Fix        string      `json:"fix,omitempty"`
	FixCommand string      `json:"fixCommand,omitempty"`
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

type CheckHostOptions struct {
	SkipVersionChecks bool
}

func CheckHost(opts CheckHostOptions) HostReport {
	r := runner.NewLocal()
	deps := HostRequiredDependencies
	if opts.SkipVersionChecks {
		deps = RemoveVersionChecks(deps)
	}
	dependencyStatuses := PerformChecks(context.Background(), deps, r)
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

func CheckTarget(ctx context.Context, dest ssh.Destination, acceptNewHostKeys bool) (TargetReport, error) {
	r, connErr := prepareRunner(ctx, dest, acceptNewHostKeys)
	status := Status{Connection: ConnectionStatus{Destination: dest, Error: connErr}}
	if connErr == nil {
		hs := ProbeHealthStatus(ctx, r)
		status.Dependencies = hs.Dependencies
		status.Hardware = hs.Hardware
	}
	return GenerateTargetReport(status), nil
}

func prepareRunner(ctx context.Context, dest ssh.Destination, acceptNewHostKeys bool) (runner.Runner, error) {
	if dest.IsPlainLocalhost() {
		return runner.NewLocal(), nil
	}
	if err := probe.SSHAuthentication(ctx, runner.NewSSH(dest), acceptNewHostKeys); err != nil {
		return nil, err
	}
	return runner.NewSSH(dest), nil
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
	remoteProcessors := targetStatus.Hardware.RemoteProcessors
	switch {
	case targetStatus.Hardware.Err != nil:
		report.SubsystemDriver.Status = CheckStatusError
		report.SubsystemDriver.Value = targetStatus.Hardware.Err.Error()
	case len(remoteProcessors) > 0:
		names := make([]string, len(remoteProcessors))
		for i, remoteProc := range remoteProcessors {
			names[i] = remoteProc.Name
		}
		report.SubsystemDriver.Status = CheckStatusOK
		report.SubsystemDriver.Value = strings.Join(names, ", ")
	default:
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
	case errors.Is(status.Error, probe.ErrAuthFailed):
		check.FixCommand = fmt.Sprintf("topo setup-keys --target %s", status.Destination)
		check.Fix = fmt.Sprintf("run `%s` to configure ssh keys", check.FixCommand)
	case errors.Is(status.Error, probe.ErrHostKeyUnknown):
		check.FixCommand = fmt.Sprintf("topo health --target %s --accept-new-host-keys", status.Destination)
		check.Fix = fmt.Sprintf("run `%s` to trust the target's identity", check.FixCommand)
	case errors.Is(status.Error, probe.ErrHostKeyChanged):
		check.FixCommand = fmt.Sprintf("ssh-keygen -R %s", status.Destination.Host)
		check.Fix = fmt.Sprintf("run `%s` to remove the old host key, then retry", check.FixCommand)
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
			} else if _, ok := errors.AsType[InfoError](ds.Error); ok {
				hc.Status = CheckStatusInfo
			} else {
				hc.Status = CheckStatusError
			}
			hc.Value = ds.Error.Error()
			hc.Fix = ds.Fix.Text
			hc.FixCommand = ds.Fix.Command
		}
		res = append(res, hc)
	}
	return res
}
