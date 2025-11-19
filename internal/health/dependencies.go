package health

import (
	"fmt"
	"os/exec"
	"regexp"
)

var HostRequiredDependencies = []Dependency{
	{Name: "ssh", Category: "SSH"},
	{Name: "docker", Category: "Container Engine"},
	{Name: "podman", Category: "Container Engine"},
}

var TargetRequiredDependencies = []Dependency{
	{Name: "docker", Category: "Container Engine"},
	{Name: "podman", Category: "Container Engine"},
}

type Dependency struct {
	Name     string
	Category string
}

type DependencyStatus struct {
	Dependency Dependency
	Installed  bool
}

var BinaryRegex = regexp.MustCompile(`^[A-Za-z0-9_+-]+$`)

type LookPath = func(bin string) (bool, error)

func CheckInstalled(dependencies []Dependency, binaryExists LookPath) []DependencyStatus {
	res := make([]DependencyStatus, len(dependencies))

	for i, dep := range dependencies {
		installed, _ := binaryExists(dep.Name)

		res[i] = DependencyStatus{
			Dependency: dep,
			Installed:  installed,
		}
	}
	return res
}

func BinaryExistsLocally(bin string) (bool, error) {
	if !BinaryRegex.MatchString(bin) {
		return false, fmt.Errorf("%q is not a valid binary name (contains invalid characters)", bin)
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
