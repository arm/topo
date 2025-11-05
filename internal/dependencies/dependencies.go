package dependencies

import "os/exec"

var RequiredDependencies = []Dependency{
	{Name: "ssh", Category: "SSH"},
	{Name: "docker", Category: "Container Engine"},
	{Name: "podman", Category: "Container Engine"},
}

type Dependency struct {
	Name     string
	Category string
}

type Status struct {
	Dependency Dependency
	Installed  bool
}

type LookPath = func(bin string) bool

func Check(dependencies []Dependency, binaryExists LookPath) []Status {
	res := make([]Status, len(dependencies))

	for i, dep := range dependencies {
		res[i] = Status{
			Dependency: dep,
			Installed:  binaryExists(dep.Name),
		}
	}

	return res
}

func BinaryExistsLocally(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func CollectAvailableByCategory(dependencyStatuses []Status) map[string][]Status {
	groupedByCategory := map[string][]Status{}

	for _, status := range dependencyStatuses {
		statuses := groupedByCategory[status.Dependency.Category]
		if status.Installed {
			statuses = append(statuses, status)
		}
		groupedByCategory[status.Dependency.Category] = statuses
	}

	return groupedByCategory
}
