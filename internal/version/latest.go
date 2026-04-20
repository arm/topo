package version

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/arm/topo/internal/output/logger"
)

const ArtifactoryBaseURL = "https://artifacts.tools.arm.com/topo"

var semverRe = regexp.MustCompile(`v(\d+)\.(\d+)\.(\d+)`)

func FetchLatest(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
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
	return versionsList[len(versionsList)-1], nil
}

func compareSemver(a, b string) int {
	pa := semverRe.FindStringSubmatch(a)
	pb := semverRe.FindStringSubmatch(b)
	for i := 1; i <= 3; i++ {
		na, _ := strconv.Atoi(pa[i])
		nb, _ := strconv.Atoi(pb[i])
		if na != nb {
			return na - nb
		}
	}
	return strings.Compare(a, b)
}
