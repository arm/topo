package version

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/arm/topo/cli/internal/output/logger"
)

const HomebrewFormulaURL = "https://raw.githubusercontent.com/arm/homebrew-topo/main/Formula/topo.rb"

var homebrewFormulaVersionRe = regexp.MustCompile(`(?m)^\s*version\s+"([^"]+)"\s*$`)

func FetchLatestHomebrew(ctx context.Context, formulaURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, formulaURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating Homebrew formula request: %w", err)
	}

	// #nosec G704 -- request to a hardcoded, trusted URL
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching Homebrew formula: %w", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.Error("failed to close Homebrew formula response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching Homebrew formula: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading Homebrew formula: %w", err)
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
