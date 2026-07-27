package version

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"

	"github.com/arm/topo/internal/fetch"
)

const ArtifactoryBaseURL = "https://artifacts.tools.arm.com/topo"

var artifactoryVersionRe = regexp.MustCompile(`href="v?(\d+)\.(\d+)\.(\d+)/"`)

func FetchLatestArtifactory(ctx context.Context, url string) (string, error) {
	body, err := fetch.Get(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetching version index: %w", err)
	}

	matches := artifactoryVersionRe.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no versions found in %q", url)
	}

	versions := make(map[string]struct{})
	for _, m := range matches {
		v := fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3])
		if _, ok := versions[v]; !ok {
			versions[v] = struct{}{}
		}
	}

	versionsList := slices.Collect(maps.Keys(versions))
	sort.Slice(versionsList, func(i, j int) bool {
		return compareSemver(versionsList[i], versionsList[j]) < 0
	})
	latest := versionsList[len(versionsList)-1]

	return latest, nil
}
