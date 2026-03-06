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

func WithCompatibility(repos []Repo) []RepoWithCompatibility {
	withCompatibility := make([]RepoWithCompatibility, len(repos))
	for i, repo := range repos {
		withCompatibility[i] = RepoWithCompatibility{Repo: repo}
	}
	return withCompatibility
}

func AnnotateCompatibility(profile target.HardwareProfile, repos []RepoWithCompatibility) []RepoWithCompatibility {
	annotated := make([]RepoWithCompatibility, len(repos))
	copy(annotated, repos)

	supportedFeatures := extractSupportedFeatures(profile)

	for r := range annotated {
		repo := &annotated[r]
		repo.Compatibility = compatibilityStatus(profile, supportedFeatures, repo.Repo)
	}

	return annotated
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
	for _, feature := range repo.Features {
		normalized := strings.ToLower(strings.TrimSpace(feature))
		if _, ok := supportedFeatures[normalized]; !ok {
			return false
		}
	}

	if repo.MinRAMKb > 0 && profile.TotalMemoryKb < repo.MinRAMKb {
		return false
	}

	return true
}
