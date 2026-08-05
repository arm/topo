package version

import (
	"context"
	"fmt"
	"regexp"

	"github.com/arm/topo/internal/fetch"
)

const HomebrewFormulaURL = "https://raw.githubusercontent.com/arm/homebrew-topo/main/Formula/topo.rb"

var homebrewFormulaVersionRe = regexp.MustCompile(`(?m)^\s*version\s+"([^"]+)"\s*$`)

func FetchLatestHomebrew(ctx context.Context, formulaURL string) (string, error) {
	body, err := fetch.Get(ctx, formulaURL)
	if err != nil {
		return "", fmt.Errorf("fetching Homebrew formula: %w", err)
	}

	return ParseHomebrewFormulaVersion(string(body))
}

func ParseHomebrewFormulaVersion(formula string) (string, error) {
	match := homebrewFormulaVersionRe.FindStringSubmatch(formula)
	if match == nil {
		return "", fmt.Errorf("no version found in Homebrew formula")
	}

	return match[1], nil
}
