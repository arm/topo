package catalog

import (
	"strings"

	"github.com/arm/topo/internal/probe"
)

type CompatibilityStatus string

const (
	CompatibilityUnknown     CompatibilityStatus = ""
	CompatibilitySupported   CompatibilityStatus = "supported"
	CompatibilityUnsupported CompatibilityStatus = "unsupported"
)

type ProjectWithCompatibility struct {
	Project
	Compatibility CompatibilityStatus `json:"compatibility,omitempty"`
}

func AnnotateCompatibility(profile *probe.HardwareProfile, projects []Project) []ProjectWithCompatibility {
	if profile == nil {
		return withCompatibility(projects)
	}

	hardwareProfile := *profile
	supportedFeatures := extractSupportedFeatures(hardwareProfile)

	checked := make([]ProjectWithCompatibility, len(projects))
	for i, project := range projects {
		checked[i].Project = project
		checked[i].Compatibility = compatibilityStatus(hardwareProfile, supportedFeatures, project)
	}

	return checked
}

func withCompatibility(projects []Project) []ProjectWithCompatibility {
	withCompatibility := make([]ProjectWithCompatibility, len(projects))
	for i, project := range projects {
		withCompatibility[i] = ProjectWithCompatibility{Project: project}
	}
	return withCompatibility
}

func compatibilityStatus(profile probe.HardwareProfile, supportedFeatures map[string]struct{}, project Project) CompatibilityStatus {
	if isProjectSupported(profile, supportedFeatures, project) {
		return CompatibilitySupported
	}
	return CompatibilityUnsupported
}

func extractSupportedFeatures(profile probe.HardwareProfile) map[string]struct{} {
	supportedFeatures := map[string]struct{}{}
	for _, proc := range profile.HostProcessors {
		for _, feature := range proc.ExtractArmFeatures() {
			supportedFeatures[strings.ToLower(feature)] = struct{}{}
		}
	}
	if len(profile.RemoteProcessors) > 0 {
		supportedFeatures["remoteproc"] = struct{}{}
		supportedFeatures["remoteproc-runtime"] = struct{}{}
	}
	return supportedFeatures
}

func isProjectSupported(profile probe.HardwareProfile, supportedFeatures map[string]struct{}, project Project) bool {
	atLeastOneFeatureIsSupported := len(project.Features) == 0

	for _, feature := range project.Features {
		normalized := strings.ToLower(strings.TrimSpace(feature))
		if _, ok := supportedFeatures[normalized]; ok {
			atLeastOneFeatureIsSupported = true
			break
		}
	}

	return atLeastOneFeatureIsSupported
}
