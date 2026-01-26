package health

import (
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/ssh"
)

type HardwareCapability int

const (
	Remoteproc HardwareCapability = iota
)

type SoftwareDependency int

const (
	UnsetSoftwareDependency SoftwareDependency = iota
	Docker
	Podman
)

type Dependency struct {
	Name                  string
	Category              string
	SoftwareEnumID        SoftwareDependency
	SoftwarePrerequisites []SoftwareDependency
	HardwarePrerequisite  []HardwareCapability
}

var HostRequiredDependencies = []Dependency{
	{Name: "ssh", Category: "SSH"},
	{Name: "docker", Category: "Container Engine", SoftwareEnumID: Docker},
	{Name: "podman", Category: "Container Engine", SoftwareEnumID: Podman},
}

var TargetRequiredDependencies = []Dependency{
	{Name: "docker", Category: "Container Engine", SoftwareEnumID: Docker},
	{Name: "podman", Category: "Container Engine", SoftwareEnumID: Podman},
	{Name: "remoteproc-runtime", Category: "Remoteproc Runtime", SoftwarePrerequisites: []SoftwareDependency{Docker, Podman}, HardwarePrerequisite: []HardwareCapability{Remoteproc}},
	{Name: "containerd-shim-remoteproc-v1", Category: "Remoteproc Shim", SoftwarePrerequisites: []SoftwareDependency{Docker}, HardwarePrerequisite: []HardwareCapability{Remoteproc}},
}

type DependencyStatus struct {
	Dependency Dependency
	Installed  bool
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

type LookPath = func(bin string) (bool, error)

func CheckInstalled(dependencies []Dependency, binaryExists LookPath) []DependencyStatus {
	installed := make(map[SoftwareDependency]struct{})
	result := make([]DependencyStatus, 0, len(dependencies))

	for _, dep := range dependencies {
		if len(dep.SoftwarePrerequisites) > 0 && !hasAnyInstalledPrerequisite(dep.SoftwarePrerequisites, installed) {
			continue
		}

		isInstalled, _ := binaryExists(dep.Name)

		if isInstalled && dep.SoftwareEnumID != UnsetSoftwareDependency {
			installed[dep.SoftwareEnumID] = struct{}{}
		}

		result = append(result, DependencyStatus{
			Dependency: dep,
			Installed:  isInstalled,
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

func BinaryExistsLocally(bin string) (bool, error) {
	if err := ssh.ValidateBinaryName(bin); err != nil {
		return false, err
	}
	_, err := exec.LookPath(bin)
	return err == nil, nil
}

func CollectAvailableByCategory(dependencyStatuses []DependencyStatus) map[string][]DependencyStatus {
	groupedByCategory := map[string][]DependencyStatus{}

	for _, status := range dependencyStatuses {
		statuses := groupedByCategory[status.Dependency.Category]
		if status.Installed {
			statuses = append(statuses, status)
		}
		groupedByCategory[status.Dependency.Category] = statuses
	}

	return groupedByCategory
}
