package health

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/arm/topo/internal/command"
)

type CheckKind int

const (
	CheckBinaryExists CheckKind = iota
	CheckCommandSuccessful
)

type CheckSeverity int

const (
	SeverityError CheckSeverity = iota
	SeverityWarning
)

type WarningError struct{ Err error }

func (w WarningError) Error() string { return w.Err.Error() }

type Check struct {
	Kind     CheckKind
	Arg      string
	Severity CheckSeverity
	Fix      string
}

func BinaryExists() Check {
	return Check{Kind: CheckBinaryExists, Severity: SeverityError}
}

func BinaryExistsWarning() Check {
	return Check{Kind: CheckBinaryExists, Severity: SeverityWarning}
}

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
		Checks: []Check{BinaryExists()},
	},
	{
		Binary:         "docker",
		Label:          "Container Engine",
		SoftwareEnumID: Docker,
		Checks: []Check{BinaryExists(), {
			Kind:     CheckCommandSuccessful,
			Arg:      "docker info",
			Severity: SeverityError,
			Fix:      "Ensure current user can run docker commands",
		}},
	},
}

var TargetRequiredDependencies = []Dependency{
	{
		Binary:         "docker",
		Label:          "Container Engine",
		SoftwareEnumID: Docker,
		Checks: []Check{BinaryExists(), {
			Kind:     CheckCommandSuccessful,
			Arg:      "docker info",
			Severity: SeverityError,
			Fix:      "Ensure current user can run docker commands",
		}},
	},
	{
		Binary:                "remoteproc-runtime",
		Label:                 "Remoteproc Runtime",
		SoftwarePrerequisites: []SoftwareDependency{Docker},
		HardwarePrerequisite:  []HardwareCapability{Remoteproc},
		Checks: []Check{
			{
				Kind:     CheckBinaryExists,
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
			{
				Kind:     CheckBinaryExists,
				Severity: SeverityWarning,
				Fix:      "run `topo install remoteproc-runtime`",
			},
		},
	},
	{
		Binary:         "lscpu",
		Label:          "Hardware Info",
		SoftwareEnumID: Lscpu,
		Checks:         []Check{BinaryExists()},
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

type (
	BinaryExistsFn      = func(bin string) error
	CommandSuccessfulFn = func(fullCmd string) error
)

func PerformChecks(dependencies []Dependency, binaryExists BinaryExistsFn, commandSuccessful CommandSuccessfulFn) []DependencyStatus {
	installed := make(map[SoftwareDependency]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))

	for _, dep := range dependencies {
		if len(dep.SoftwarePrerequisites) > 0 && !hasAnyInstalledPrerequisite(dep.SoftwarePrerequisites, installed) {
			continue
		}

		var err error
		var fix string
		for _, check := range dep.Checks {
			switch check.Kind {
			case CheckBinaryExists:
				err = binaryExists(dep.Binary)
				if err == nil && dep.SoftwareEnumID != UnsetSoftwareDependency {
					installed[dep.SoftwareEnumID] = struct{}{}
				}
			case CheckCommandSuccessful:
				err = commandSuccessful(check.Arg)
			}
			if err != nil {
				if check.Severity == SeverityWarning {
					err = WarningError{Err: err}
				}
				fix = check.Fix
				break
			}
		}

		result = append(result, DependencyStatus{
			Dependency: dep,
			Error:      err,
			Fix:        fix,
		})
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

func BinaryExistsLocally(bin string) error {
	if err := command.ValidateBinaryName(bin); err != nil {
		return err
	}
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%q executable file not found in $PATH", bin)
	}
	return nil
}

func CommandSuccessfulLocally(fullCmd string) error {
	cmdParts := strings.Fields(fullCmd)
	execCmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("failed to run `%s`: %w", fullCmd, err)
	}
	return nil
}
