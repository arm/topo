package catalog

import (
	"strings"

	"github.com/arm/topo/internal/target"
)

type Compatibility struct {
	Supported bool `json:"supported"`
}

type RepoWithCompatibility struct {
	Repo
	Compatibility *Compatibility `json:"compatibility,omitempty"`
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

	for r := range annotated {
		repo := &annotated[r]
		supported := true

		for _, feature := range repo.Features {
			normalized := strings.ToLower(strings.TrimSpace(feature))
			if _, ok := supportedFeatures[normalized]; !ok {
				supported = false
				break
			}
		}

		if repo.MinRAMKb > 0 {
			if profile.TotalMemoryKb == 0 {
				supported = false
			} else if profile.TotalMemoryKb < repo.MinRAMKb {
				supported = false
			}
		}

		repo.Compatibility = &Compatibility{Supported: supported}
	}

	return annotated
}
