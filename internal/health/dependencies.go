package health

import (
	"fmt"
	"os/exec"

	"github.com/arm/topo/internal/ssh"
)

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
	SoftwareEnumID        SoftwareDependency
	SoftwarePrerequisites []SoftwareDependency
	HardwarePrerequisite  []HardwareCapability
}

var HostRequiredDependencies = []Dependency{
	{Binary: "ssh", Label: "SSH"},
	{Binary: "docker", Label: "Container Engine", SoftwareEnumID: Docker},
}

var TargetRequiredDependencies = []Dependency{
	{Binary: "docker", Label: "Container Engine", SoftwareEnumID: Docker},
	{Binary: "remoteproc-runtime", Label: "Remoteproc Runtime", SoftwarePrerequisites: []SoftwareDependency{Docker}, HardwarePrerequisite: []HardwareCapability{Remoteproc}},
	{Binary: "containerd-shim-remoteproc-v1", Label: "Remoteproc Shim", SoftwarePrerequisites: []SoftwareDependency{Docker}, HardwarePrerequisite: []HardwareCapability{Remoteproc}},
	{Binary: "lscpu", Label: "Hardware Info", SoftwareEnumID: Lscpu},
}

type DependencyStatus struct {
	Dependency Dependency
	Error      error
}

func CheckDependencies(binaryExists func(string) error, capabilities map[HardwareCapability]struct{}) []DependencyStatus {
	deps := FilterByHardware(TargetRequiredDependencies, capabilities)
	return CheckInstalled(deps, binaryExists)
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

func CheckInstalled(dependencies []Dependency, binaryExists BinaryExistsFn) []DependencyStatus {
	installed := make(map[SoftwareDependency]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))

	for _, dep := range dependencies {
		if len(dep.SoftwarePrerequisites) > 0 && !hasAnyInstalledPrerequisite(dep.SoftwarePrerequisites, installed) {
			continue
		}

		err := binaryExists(dep.Binary)

		if err == nil && dep.SoftwareEnumID != UnsetSoftwareDependency {
			installed[dep.SoftwareEnumID] = struct{}{}
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
