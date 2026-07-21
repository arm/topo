package version

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"sort"

	"github.com/arm/topo/internal/output/logger"
)

const ArtifactoryBaseURL = "https://artifacts.tools.arm.com/topo"

func FetchLatestArtifactory(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	// #nosec G704 -- request to a hardcoded, trusted URL
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching version index: %w", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.Error("failed to close version check response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching version index: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading version index: %w", err)
	}

	matches := semverRe.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no versions found in %q", url)
	}

	versions := make(map[string]struct{})
	for _, m := range matches {
		v := m[0]
		if _, ok := versions[v]; !ok {
			versions[v] = struct{}{}
		}
	}

	versionsList := slices.Collect(maps.Keys(versions))
	sort.Slice(versionsList, func(i, j int) bool {
		return compareSemver(versionsList[i], versionsList[j]) < 0
	})
	latest := versionsList[len(versionsList)-1]

	return latest[1:], nil
}
