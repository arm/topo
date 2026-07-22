package install

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	archiveutil "github.com/arm/topo/internal/archive"
	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/version"
)

const (
	downloadTimeout = 2 * time.Minute
)

var defaultCandidatePaths = []string{"/usr/local/bin", "/usr/bin", "~/bin"}

type PathCandidate struct {
	Path   string
	OnPath bool
}

func getPathDirs(r runner.Runner) ([]string, error) {
	output, _, err := r.Run(context.TODO(), "echo $PATH")
	if err != nil {
		return nil, err
	}

	pathStr := strings.TrimSpace(output)
	paths := strings.Split(pathStr, ":")

	return paths, nil
}

func getHomeDir(r runner.Runner) (string, error) {
	output, _, err := r.Run(context.TODO(), "echo $HOME")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func getExistingBinaryDir(r runner.Runner, binaryName string) (string, error) {
	checkCommand, err := command.BinaryLookupCommand(binaryName)
	if err != nil {
		return "", err
	}

	output, _, err := r.Run(context.TODO(), checkCommand)
	if err != nil {
		return "", nil
	}

	fullPath := strings.TrimSpace(output)
	if fullPath == "" {
		return "", nil
	}

	lastSlash := strings.LastIndex(fullPath, "/")
	if lastSlash == -1 {
		return "", fmt.Errorf("invalid path format: %s", fullPath)
	}

	return fullPath[:lastSlash], nil
}

func FindPathDirs(r runner.Runner) ([]PathCandidate, error) {
	pathDirs, err := getPathDirs(r)
	if err != nil {
		return nil, err
	}

	homeDir, err := getHomeDir(r)
	if err != nil {
		return nil, err
	}

	var validPaths []PathCandidate
	for _, candidate := range defaultCandidatePaths {
		expanded := candidate
		if strings.HasPrefix(candidate, "~/") {
			expanded = homeDir + candidate[1:]
		}
		validPaths = append(validPaths, PathCandidate{
			Path:   expanded,
			OnPath: slices.Contains(pathDirs, expanded),
		})
	}

	return validPaths, nil
}

func downloadFile(ctx context.Context, url *url.URL) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	// #nosec G704 -- Request is previously validated
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected fetch status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func install(installPath string, r runner.Runner, binaries map[string][]byte) error {
	mode := "0755"

	for binaryName, binaryData := range binaries {
		if err := command.ValidateBinaryName(binaryName); err != nil {
			return err
		}

		installCmd := fmt.Sprintf("install -D -m %s /dev/stdin %s/%s", mode, installPath, binaryName)
		_, stderr, err := r.RunWithStdin(context.TODO(), installCmd, binaryData)
		if err != nil {
			if errors.Is(err, ssh.ErrSSH) {
				return err
			}
			if classified := classifyStderr(stderr); classified != nil {
				return classified
			}
			return err
		}
	}
	return nil
}

// installToFirstWriteableDir attempts to install binaries to the highest preference path that the user has permissions for.
// Silently ignores permission failures until the last path.
// Returns the installation location and a list of installed binary names.
func installToFirstWriteableDir(paths []PathCandidate, r runner.Runner, binaries map[string][]byte) (PathCandidate, []string, error) {
	var binaryNames []string
	for name := range binaries {
		binaryNames = append(binaryNames, name)
	}

	for i, dir := range paths {
		err := install(dir.Path, r, binaries)
		if err == nil {
			return paths[i], binaryNames, nil
		}
		if !errors.Is(err, ErrPermissionDenied) {
			return PathCandidate{}, nil, err
		}
	}
	candidatePaths := make([]string, len(paths))
	for i, p := range paths {
		candidatePaths[i] = p.Path
	}
	return PathCandidate{}, nil, fmt.Errorf("permission denied for all candidate directories: %v", candidatePaths)
}

type InstallResult struct {
	Location PathCandidate
	Binary   string
}

func downloadLatestArtifactoryBinaries(ctx context.Context, artifactoryURL string, archiveNameFormat string, binaryNames []string) (map[string][]byte, error) {
	latest, err := version.FetchLatestArtifactory(ctx, artifactoryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve latest Artifactory version: %w", err)
	}

	archiveName := fmt.Sprintf(archiveNameFormat, latest)
	archiveURL, err := url.JoinPath(artifactoryURL, latest, archiveName)
	if err != nil {
		return nil, fmt.Errorf("failed to construct Artifactory download URL: %w", err)
	}

	parsedArchiveURL, err := url.Parse(archiveURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Artifactory download URL: %w", err)
	}

	tarball, err := downloadFile(ctx, parsedArchiveURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download latest release: %w", err)
	}

	binaries, err := archiveutil.ExtractFiles(ctx, tarball, binaryNames)
	if err != nil {
		return nil, fmt.Errorf("failed to extract files from tar.gz: %w", err)
	}
	return binaries, nil
}

func InstallBinariesFromArtifactory(ctx context.Context, r runner.Runner, artifactoryURL string, archiveNameFormat string, binaryNames []string) ([]InstallResult, error) {
	for _, binaryName := range binaryNames {
		if err := command.ValidateBinaryName(binaryName); err != nil {
			return nil, err
		}
	}

	binaries, err := downloadLatestArtifactoryBinaries(ctx, artifactoryURL, archiveNameFormat, binaryNames)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release binaries: %w", err)
	}

	existingBinaryPaths := make(map[string]string)
	var binariesNotOnPath []string

	for _, binaryName := range binaryNames {
		existingPath, err := getExistingBinaryDir(r, binaryName)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing path for %s: %w", binaryName, err)
		}

		if existingPath != "" {
			existingBinaryPaths[binaryName] = existingPath
		} else {
			binariesNotOnPath = append(binariesNotOnPath, binaryName)
		}
	}

	var results []InstallResult

	for binaryName, dirPath := range existingBinaryPaths {
		binaryData, ok := binaries[binaryName]
		if !ok {
			return nil, fmt.Errorf("binary %s not found in release", binaryName)
		}

		singleBinary := map[string][]byte{binaryName: binaryData}
		err := install(dirPath, r, singleBinary)
		if err != nil {
			return nil, fmt.Errorf("failed to install %s to existing location %s: %w", binaryName, dirPath, err)
		}

		results = append(results, InstallResult{
			Location: PathCandidate{Path: dirPath, OnPath: true},
			Binary:   binaryName,
		})
	}

	// if not already on path, find a good spot for them.
	if len(binariesNotOnPath) > 0 {
		paths, err := FindPathDirs(r)
		if err != nil {
			return nil, fmt.Errorf("failed to find valid PATH directories: %w", err)
		}

		newBinariesMap := make(map[string][]byte)
		for _, binaryName := range binariesNotOnPath {
			newBinariesMap[binaryName] = binaries[binaryName]
		}

		installLoc, installedBinaries, err := installToFirstWriteableDir(paths, r, newBinariesMap)
		if err != nil {
			return nil, fmt.Errorf("installation of new binaries failed: %w", err)
		}

		// Re-check PATH after installation: on some systems (e.g. Ubuntu/Debian),
		// ~/.profile only adds ~/bin to PATH if the directory already exists.
		// Creating the directory during install means a new login shell will now
		// include the path, even though it wasn't present during the earlier check.
		if !installLoc.OnPath {
			if pathDirs, pathErr := getPathDirs(r); pathErr == nil {
				installLoc.OnPath = slices.Contains(pathDirs, installLoc.Path)
			}
		}

		for _, binaryName := range installedBinaries {
			results = append(results, InstallResult{
				Location: installLoc,
				Binary:   binaryName,
			})
		}
	}

	return results, nil
}

var ErrPermissionDenied = errors.New("permission denied")

func classifyStderr(stderr string) error {
	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "cannot create") ||
		strings.Contains(lower, "read-only") {
		return ErrPermissionDenied
	}
	return nil
}
