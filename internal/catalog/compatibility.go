package catalog

import (
	"strings"

	"github.com/arm/topo/internal/target"
)

type CompatibilityStatus string

const (
	CompatibilityUnknown     CompatibilityStatus = ""
	CompatibilitySupported   CompatibilityStatus = "supported"
	CompatibilityUnsupported CompatibilityStatus = "unsupported"
)

type RepoWithCompatibility struct {
	Repo
	Compatibility CompatibilityStatus `json:"compatibility,omitempty"`
}

func AnnotateCompatibility(profile *target.HardwareProfile, repos []Repo) []RepoWithCompatibility {
	if profile == nil {
		return withCompatibility(repos)
	}

	hardwareProfile := *profile
	supportedFeatures := extractSupportedFeatures(hardwareProfile)

	checked := make([]RepoWithCompatibility, len(repos))
	for i, repo := range repos {
		checked[i].Repo = repo
		checked[i].Compatibility = compatibilityStatus(hardwareProfile, supportedFeatures, repo)
	}

	return checked
}

func withCompatibility(repos []Repo) []RepoWithCompatibility {
	withCompatibility := make([]RepoWithCompatibility, len(repos))
	for i, repo := range repos {
		withCompatibility[i] = RepoWithCompatibility{Repo: repo}
	}
	return withCompatibility
}

func compatibilityStatus(profile target.HardwareProfile, supportedFeatures map[string]struct{}, repo Repo) CompatibilityStatus {
	if isRepoSupported(profile, supportedFeatures, repo) {
		return CompatibilitySupported
	}
	return CompatibilityUnsupported
}

func extractSupportedFeatures(profile target.HardwareProfile) map[string]struct{} {
	supportedFeatures := map[string]struct{}{}
	for _, proc := range profile.HostProcessor {
		for _, feature := range proc.ExtractArmFeatures() {
			supportedFeatures[strings.ToLower(feature)] = struct{}{}
		}
	}
	if len(profile.RemoteCPU) > 0 {
		supportedFeatures["remoteproc"] = struct{}{}
		supportedFeatures["remoteproc-runtime"] = struct{}{}
	}
	return supportedFeatures
}

func isRepoSupported(profile target.HardwareProfile, supportedFeatures map[string]struct{}, repo Repo) bool {
	if repo.MinRAMKb > 0 && profile.TotalMemoryKb < repo.MinRAMKb {
		return false
	}

	atLeastOneFeatureIsSupported := len(repo.Features) == 0

	for _, feature := range repo.Features {
		normalized := strings.ToLower(strings.TrimSpace(feature))
		if _, ok := supportedFeatures[normalized]; ok {
			atLeastOneFeatureIsSupported = true
			break
		}
	}

	return atLeastOneFeatureIsSupported
}
