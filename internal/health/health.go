package health

import "context"

type CheckStatus string

const (
	CheckStatusOK           CheckStatus = "ok"
	CheckStatusWarning      CheckStatus = "warning"
	CheckStatusError        CheckStatus = "error"
	CheckStatusInfo         CheckStatus = "info"
	CheckStatusUndetermined CheckStatus = "undetermined"
)

type DependencyReport struct {
	Scope     DependencyScope
	ID        DependencyID
	Name      string
	Status    CheckStatus
	Value     string
	Fix       *Fix
	BlockedBy []DependencyBlocker
}

type DependencyBlocker struct {
	Scope DependencyScope
	Name  string
}

type TargetDetails struct {
	Destination string
	IsLocalhost bool
}

type ReadinessReport struct {
	Checks       []DependencyReport
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
	deployment := ReadinessReport{Checks: toDependencyReports(h.Deployment.Dependencies)}
	discovery := ReadinessReport{Checks: toDependencyReports(h.ProjectDiscovery.Dependencies)}
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

func ToDependencyReport(dependency EvaluatedDependency) DependencyReport {
	report := DependencyReport{Scope: dependency.Scope, ID: dependency.ID, Name: dependency.Label}
	switch dependency.Evaluation.State {
	case EvaluationBlocked:
		report.Status = CheckStatusUndetermined
		report.BlockedBy = dependencyBlockers(dependency.Evaluation.BlockedBy)
		return report
	case EvaluationOmitted:
		panic("cannot report an omitted health dependency")
	case EvaluationExecuted:
	default:
		panic("unknown health dependency evaluation state")
	}

	result := dependency.Evaluation.Result
	if result.Failure == nil {
		report.Status = CheckStatusOK
		report.Value = result.SuccessValue
		return report
	}

	report.Status = checkStatusFromSeverity(result.Failure.Severity)
	report.Value = result.Failure.Message
	report.Fix = result.Failure.Fix
	return report
}

func dependencyBlockers(blockers []*DependencyNode) []DependencyBlocker {
	references := make([]DependencyBlocker, len(blockers))
	for i, blocker := range blockers {
		references[i] = DependencyBlocker{
			Scope: blocker.Scope(),
			Name:  blocker.Dependency().Label,
		}
	}
	return references
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
