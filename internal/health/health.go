package health

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/arm/topo/internal/command"
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
	Name   string
	Status CheckStatus
	Value  string
	Fix    *Fix
}

type HostReport struct {
	Dependencies []HealthCheck
}

type TargetReport struct {
	Destination            string
	IsLocalhost            bool
	Connectivity           HealthCheck
	Dependencies           []HealthCheck
	ProcessingDomainDriver HealthCheck
}

type CheckHostOptions struct {
	SkipVersionChecks bool
}

func CheckHost(opts CheckHostOptions) HostReport {
	r := runner.NewLocal()
	deps := HostRequiredDependencies(opts.SkipVersionChecks)
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
		hs := ProbeHealthStatus(ctx, r, dest)
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

	report.ProcessingDomainDriver.Name = "Processing Domain Driver (remoteproc)"
	remoteProcessors := targetStatus.Hardware.RemoteProcessors
	switch {
	case targetStatus.Hardware.Err != nil:
		report.ProcessingDomainDriver.Status = CheckStatusError
		report.ProcessingDomainDriver.Value = targetStatus.Hardware.Err.Error()
	case len(remoteProcessors) > 0:
		names := make([]string, len(remoteProcessors))
		for i, remoteProc := range remoteProcessors {
			names[i] = remoteProc.Name
		}
		report.ProcessingDomainDriver.Status = CheckStatusOK
		report.ProcessingDomainDriver.Value = strings.Join(names, ", ")
	default:
		report.ProcessingDomainDriver.Status = CheckStatusInfo
		report.ProcessingDomainDriver.Value = "no remoteproc devices found"
	}

	report.Dependencies = generateDependencyReport(targetStatus.Dependencies)
	report.Destination = targetStatus.Connection.Destination.String()

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
	case errors.Is(status.Error, probe.ErrAuthFailed) || errors.Is(status.Error, probe.ErrTooManyAuthFails):
		check.Fix = &Fix{
			Description: "Configure SSH keys on remote target",
			Command:     fmt.Sprintf("topo setup-keys --target %s", status.Destination),
		}
	case errors.Is(status.Error, probe.ErrHostKeyUnknown):
		check.Fix = &Fix{
			Description: "Trust the target's SSH host key",
			Command:     fmt.Sprintf("topo health --target %s --accept-new-host-keys", status.Destination),
		}
	case errors.Is(status.Error, probe.ErrHostKeyChanged):
		sshConfig, err := ssh.LoadConfig(status.Destination)
		var fixCommand string
		if err == nil {
			fixCommand = fmt.Sprintf("ssh-keygen -R %s", command.QuoteArg(sshConfig.AsKnownHostsEntry()))
		}
		check.Fix = &Fix{
			Description: "Remove the old SSH host key from known_hosts, then retry",
			Command:     fixCommand,
		}
	}
	return check
}

func generateDependencyReport(statuses []DependencyStatus) []HealthCheck {
	res := []HealthCheck{}
	for _, ds := range statuses {
		hc := HealthCheck{Name: ds.Dependency.Label}
		if ds.Result.Failure == nil {
			hc.Status = CheckStatusOK
			hc.Value = ds.Result.SuccessValue
		} else {
			hc.Status = checkStatusFromSeverity(ds.Result.Failure.Severity)
			hc.Value = ds.Result.Failure.Message
			hc.Fix = ds.Result.Failure.Fix
		}
		res = append(res, hc)
	}
	return res
}

func checkStatusFromSeverity(severity CheckSeverity) CheckStatus {
	switch severity {
	case SeverityWarning:
		return CheckStatusWarning
	case SeverityInfo:
		return CheckStatusInfo
	default:
		return CheckStatusError
	}
}
