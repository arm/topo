package catalog

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
)

//go:embed data/templates.json
var TemplatesJSON []byte

type TemplateFilters struct {
	Target   string
	Features []string
}

type Repo struct {
	Id          string   `json:"id"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Url         string   `json:"url"`
	Ref         string   `json:"ref"`
}

func GetTemplateRepo(id string) (*Repo, error) {
	return GetRepo(id, TemplatesJSON)
}

func FilterTemplateRepos(flags TemplateFilters, repos []Repo) ([]Repo, error) {
	targetMode := false

	if flags.Target != "" {
		conn := target.NewConnection(flags.Target, ssh.ExecSSH, target.ConnectionOptions{})
		hw, err := conn.ProbeHardware()
		if err != nil {
			return nil, err
		}
		if len(hw.HostProcessor) > 0 {
			flags.Features = hw.HostProcessor[0].ExtractArmFeatures()
		}
		targetMode = true
	}

	var filtered []Repo
	for _, repo := range repos {
		if supportsFeatures(flags.Features, repo, targetMode) {
			filtered = append(filtered, repo)
		}
	}
	return filtered, nil
}

func supportsFeatures(features []string, repo Repo, targetMode bool) bool {
	if len(features) == 0 && !targetMode {
		return true
	}

	lowerRequested := make([]string, len(features))
	for i, f := range features {
		lowerRequested[i] = strings.ToLower(f)
	}

	lowerRepo := make([]string, len(repo.Features))
	for i, f := range repo.Features {
		lowerRepo[i] = strings.ToLower(f)
	}

	if targetMode {
		for _, feature := range lowerRepo {
			if !slices.Contains(lowerRequested, feature) {
				return false
			}
		}
		return true
	}

	for _, requested := range lowerRequested {
		if !slices.Contains(lowerRepo, requested) {
			return false
		}
	}
	return true
}

func ParseRepos(b []byte) ([]Repo, error) {
	var templates []Repo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetRepo(id string, b []byte) (*Repo, error) {
	repos, err := ParseRepos(b)
	if err != nil {
		return nil, err
	}
	for i := range repos {
		if repos[i].Id == id {
			return &repos[i], nil
		}
	}
	return nil, fmt.Errorf("repo with id %q not found", id)
}
