package health

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/runner"
)

type WarningError struct{ Err error }

func (w WarningError) Error() string { return w.Err.Error() }

type HardwareCapability int

const (
	Remoteproc HardwareCapability = iota
)

type SoftwareDependency int

const (
	UnsetSoftwareDependency SoftwareDependency = iota
	Docker
	Lscpu
)

type Dependency struct {
	Binary                string
	Label                 string
	Checks                []Check
	SoftwareEnumID        SoftwareDependency
	SoftwarePrerequisites []SoftwareDependency
	HardwarePrerequisite  []HardwareCapability
}

var HostRequiredDependencies = []Dependency{
	{
		Binary: "ssh",
		Label:  "SSH",
		Checks: []Check{BinaryExists{}},
	},
	{
		Binary:         "docker",
		Label:          "Container Engine",
		SoftwareEnumID: Docker,
		Checks: []Check{
			BinaryExists{},
			CommandSuccessful{
				Cmd: "docker info",
				Fix: "Ensure current user can run docker commands",
			},
		},
	},
}

var TargetRequiredDependencies = []Dependency{
	{
		Binary:         "docker",
		Label:          "Container Engine",
		SoftwareEnumID: Docker,
		Checks: []Check{
			BinaryExists{},
			CommandSuccessful{
				Cmd: "docker info",
				Fix: "Ensure current user can run docker commands",
			},
		},
	},
	{
		Binary:                "remoteproc-runtime",
		Label:                 "Remoteproc Runtime",
		SoftwarePrerequisites: []SoftwareDependency{Docker},
		HardwarePrerequisite:  []HardwareCapability{Remoteproc},
		Checks: []Check{
			BinaryExists{
				Severity: SeverityWarning,
				Fix:      "run `topo install remoteproc-runtime`",
			},
		},
	},
	{
		Binary:                "containerd-shim-remoteproc-v1",
		Label:                 "Remoteproc Shim",
		SoftwarePrerequisites: []SoftwareDependency{Docker},
		HardwarePrerequisite:  []HardwareCapability{Remoteproc},
		Checks: []Check{
			BinaryExists{
				Severity: SeverityWarning,
				Fix:      "run `topo install remoteproc-runtime`",
			},
		},
	},
	{
		Binary:         "lscpu",
		Label:          "Hardware Info",
		SoftwareEnumID: Lscpu,
		Checks:         []Check{BinaryExists{}},
	},
}

type DependencyStatus struct {
	Dependency Dependency
	Error      error
	Fix        string
}

func FilterByHardware(deps []Dependency, hardware map[HardwareCapability]struct{}) []Dependency {
	result := make([]Dependency, 0, len(deps))
	for _, dep := range deps {
		if len(dep.HardwarePrerequisite) == 0 || hardwareCapabilityMatches(dep.HardwarePrerequisite, hardware) {
			result = append(result, dep)
		}
	}
	return result
}

func hardwareCapabilityMatches(required []HardwareCapability, available map[HardwareCapability]struct{}) bool {
	for _, capability := range required {
		if _, exists := available[capability]; exists {
			return true
		}
	}
	return false
}

func PerformChecks(ctx context.Context, dependencies []Dependency, r runner.Runner) []DependencyStatus {
	installed := make(map[SoftwareDependency]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))
	var timedOut bool

	for _, dep := range dependencies {
		if len(dep.SoftwarePrerequisites) > 0 && !hasAnyInstalledPrerequisite(dep.SoftwarePrerequisites, installed) {
			continue
		}

		if timedOut {
			result = append(result, DependencyStatus{
				Dependency: dep,
				Error:      runner.ErrTimeout,
			})
			continue
		}

		var fix string
		var err error
		for _, check := range dep.Checks {
			fix, err = check.Run(ctx, r, dep)
			if err == nil {
				if _, ok := check.(BinaryExists); ok && dep.SoftwareEnumID != UnsetSoftwareDependency {
					installed[dep.SoftwareEnumID] = struct{}{}
				}
				continue
			}
			break
		}

		result = append(result, DependencyStatus{
			Dependency: dep,
			Error:      err,
			Fix:        fix,
		})

		if errors.Is(err, runner.ErrTimeout) {
			timedOut = true
		}
	}
	return result
}

func hasAnyInstalledPrerequisite(required []SoftwareDependency, installed map[SoftwareDependency]struct{}) bool {
	for _, softwareDep := range required {
		if _, exists := installed[softwareDep]; exists {
			return true
		}
	}
	return false
}
