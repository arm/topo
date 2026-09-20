package health

import "context"

type CheckStatus string

const (
	CheckStatusOK      CheckStatus = "ok"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusError   CheckStatus = "error"
	CheckStatusInfo    CheckStatus = "info"
)

type DependencyReport struct {
	Scope  DependencyScope
	ID     DependencyID
	Name   string
	Status CheckStatus
	Value  string
	Fix    *Fix
}

type TargetDetails struct {
	Destination string
	IsLocalhost bool
}

type ReadinessReport struct {
	Host         []DependencyReport
	Target       []DependencyReport
	TargetStatus *TargetStatus
}

type TargetStatus struct {
	Status CheckStatus
	Fix    *Fix
}

type HealthReport struct {
	TargetDetails    *TargetDetails
	Deployment       ReadinessReport
	ProjectDiscovery ReadinessReport
}

func Check(ctx context.Context, options HealthCheckOptions) HealthReport {
	healthCheck := NewHealthCheck(options)
	evaluatedHealthCheck := healthCheck.Evaluate(ctx)
	return evaluatedHealthCheck.Report(targetDetails(options), options.MissingTargetFixMessage)
}

func (h EvaluatedHealthCheck) Report(target *TargetDetails, missingTargetFixMessage string) HealthReport {
	deployment := toReadinessReport(h.Deployment)
	discovery := toReadinessReport(h.ProjectDiscovery)
	if target == nil {
		deployment.TargetStatus = missingTargetStatus(CheckStatusError, missingTargetFixMessage)
		discovery.TargetStatus = missingTargetStatus(CheckStatusWarning, missingTargetFixMessage)
	}
	return HealthReport{
		TargetDetails:    target,
		Deployment:       deployment,
		ProjectDiscovery: discovery,
	}
}

func missingTargetStatus(status CheckStatus, fixMessage string) *TargetStatus {
	targetStatus := &TargetStatus{Status: status}
	if fixMessage != "" {
		targetStatus.Fix = &Fix{Description: fixMessage}
	}
	return targetStatus
}

func targetDetails(options HealthCheckOptions) *TargetDetails {
	if options.Target == nil {
		return nil
	}
	return &TargetDetails{
		Destination: options.Target.String(),
		IsLocalhost: options.Target.IsPlainLocalhost(),
	}
}

func toReadinessReport(evaluatedReadinessCheck EvaluatedReadinessCheck) ReadinessReport {
	report := ReadinessReport{}
	for _, dependency := range toDependencyReports(evaluatedReadinessCheck.Dependencies) {
		switch dependency.Scope {
		case DependencyScopeHost:
			report.Host = append(report.Host, dependency)
		case DependencyScopeTarget:
			report.Target = append(report.Target, dependency)
		default:
			panic("health dependency has an unknown scope")
		}
	}
	return report
}

func ToDependencyReport(status EvaluatedDependency) DependencyReport {
	report := DependencyReport{Scope: status.Scope, ID: status.ID, Name: status.Label}
	if status.Result.Failure == nil {
		report.Status = CheckStatusOK
		report.Value = status.Result.SuccessValue
		return report
	}

	report.Status = checkStatusFromSeverity(status.Result.Failure.Severity)
	report.Value = status.Result.Failure.Message
	report.Fix = status.Result.Failure.Fix
	return report
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

func toDependencyReports(statuses []EvaluatedDependency) []DependencyReport {
	reports := make([]DependencyReport, len(statuses))
	for i, status := range statuses {
		reports[i] = ToDependencyReport(status)
	}
	return reports
}
