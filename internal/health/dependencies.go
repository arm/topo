package health

import (
	"fmt"
	"os/exec"

	"github.com/arm/topo/internal/collections"
	"github.com/arm/topo/internal/ssh"
)

type CheckKind int

const (
	CheckBinaryExists CheckKind = iota
)

type Check struct {
	Kind CheckKind
	Arg  string
}

func BinaryExists(bin string) Check {
	return Check{Kind: CheckBinaryExists, Arg: bin}
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
	Name                  string
	Category              string
	Checks                []Check
	SoftwareEnumID        SoftwareDependency
	SoftwarePrerequisites []SoftwareDependency
	HardwarePrerequisite  []HardwareCapability
}

var HostRequiredDependencies = []Dependency{
	{
		Name:     "ssh",
		Category: "SSH",
		Checks:   []Check{BinaryExists("ssh")},
	},
	{
		Name:           "docker",
		Category:       "Container Engine",
		SoftwareEnumID: Docker,
		Checks:         []Check{BinaryExists("docker")},
	},
}

var TargetRequiredDependencies = []Dependency{
	{
		Name:           "docker",
		Category:       "Container Engine",
		SoftwareEnumID: Docker,
		Checks:         []Check{BinaryExists("docker")},
	},
	{
		Name:                  "remoteproc-runtime",
		Category:              "Remoteproc Runtime",
		SoftwarePrerequisites: []SoftwareDependency{Docker},
		HardwarePrerequisite:  []HardwareCapability{Remoteproc},
		Checks:                []Check{BinaryExists("remoteproc-runtime")},
	},
	{
		Name:                  "containerd-shim-remoteproc-v1",
		Category:              "Remoteproc Shim",
		SoftwarePrerequisites: []SoftwareDependency{Docker},
		HardwarePrerequisite:  []HardwareCapability{Remoteproc},
		Checks:                []Check{BinaryExists("containerd-shim-remoteproc-v1")},
	},
	{
		Name:           "lscpu",
		Category:       "Hardware Info",
		SoftwareEnumID: Lscpu,
		Checks:         []Check{BinaryExists("lscpu")},
	},
}

type DependencyStatus struct {
	Dependency Dependency
	Error      error
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

type BinaryExistsFn = func(bin string) error

func PerformChecks(dependencies []Dependency, binaryExists BinaryExistsFn) []DependencyStatus {
	installed := make(map[SoftwareDependency]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))

	for _, dep := range dependencies {
		if len(dep.SoftwarePrerequisites) > 0 && !hasAnyInstalledPrerequisite(dep.SoftwarePrerequisites, installed) {
			continue
		}

		var err error
		for _, check := range dep.Checks {
			switch check.Kind {
			case CheckBinaryExists:
				err = binaryExists(check.Arg)
				if err == nil && dep.SoftwareEnumID != UnsetSoftwareDependency {
					installed[dep.SoftwareEnumID] = struct{}{}
				}
			}
			if err != nil {
				break
			}
		}

		result = append(result, DependencyStatus{
			Dependency: dep,
			Error:      err,
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
	if err := ssh.ValidateBinaryName(bin); err != nil {
		return err
	}
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%q executable file not found in $PATH", bin)
	}
	return nil
}

func groupByCategory(statuses []DependencyStatus) []collections.Group[DependencyStatus, string] {
	return collections.GroupBy(
		statuses,
		func(ds DependencyStatus) string { return ds.Dependency.Category },
	)
}
