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

type TemplateWithCompatibility struct {
	Template
	Compatibility CompatibilityStatus `json:"compatibility,omitempty"`
}

func AnnotateCompatibility(profile *probe.HardwareProfile, templates []Template) []TemplateWithCompatibility {
	if profile == nil {
		return withCompatibility(templates)
	}

	hardwareProfile := *profile
	supportedFeatures := extractSupportedFeatures(hardwareProfile)

	checked := make([]TemplateWithCompatibility, len(templates))
	for i, template := range templates {
		checked[i].Template = template
		checked[i].Compatibility = compatibilityStatus(hardwareProfile, supportedFeatures, template)
	}

	return checked
}

func withCompatibility(templates []Template) []TemplateWithCompatibility {
	withCompatibility := make([]TemplateWithCompatibility, len(templates))
	for i, template := range templates {
		withCompatibility[i] = TemplateWithCompatibility{Template: template}
	}
	return withCompatibility
}

func compatibilityStatus(profile probe.HardwareProfile, supportedFeatures map[string]struct{}, template Template) CompatibilityStatus {
	if isTemplateSupported(profile, supportedFeatures, template) {
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

func isTemplateSupported(profile probe.HardwareProfile, supportedFeatures map[string]struct{}, template Template) bool {
	atLeastOneFeatureIsSupported := len(template.Features) == 0

	for _, feature := range template.Features {
		normalized := strings.ToLower(strings.TrimSpace(feature))
		if _, ok := supportedFeatures[normalized]; ok {
			atLeastOneFeatureIsSupported = true
			break
		}
	}

	return atLeastOneFeatureIsSupported
}
