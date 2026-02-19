package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/arm/topo/internal/ssh"
	"github.com/mholt/archives"
)

const (
	apiTimeout      = 15 * time.Second
	downloadTimeout = 2 * time.Minute
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrSSHAuthFailed    = errors.New("SSH authentication failed")
)

var defaultCandidatePaths = []string{"/usr/local/bin", "/usr/bin", "~/bin"}

type PathCandidate struct {
	Path   string
	OnPath bool
}

func getPathDirs(targetHost ssh.Host) ([]string, error) {
	output, err := ssh.ExecWithShell(targetHost, "echo $PATH")
	if err != nil {
		return nil, err
	}

	pathStr := strings.TrimSpace(output)
	paths := strings.Split(pathStr, ":")

	return paths, nil
}

func getHomeDir(targetHost ssh.Host) (string, error) {
	output, err := ssh.ExecWithShell(targetHost, "echo $HOME")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func getExistingBinaryDir(targetHost ssh.Host, binaryName string) (string, error) {
	output, err := ssh.ExecWithShell(targetHost, fmt.Sprintf("command -v %s", binaryName))
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

func FindPathDirs(targetHost ssh.Host) ([]PathCandidate, error) {
	pathDirs, err := getPathDirs(targetHost)
	if err != nil {
		return nil, err
	}

	homeDir, err := getHomeDir(targetHost)
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

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	Assets []githubAsset `json:"assets"`
}

func addGitHubAuthHeader(req *http.Request) {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		return
	}

	host := req.URL.Hostname()
	if host == "github.com" || host == "api.github.com" || strings.HasSuffix(host, ".github.com") {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func getLatestReleaseTarAddress(repoURL string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoURL)

	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	addGitHubAuthHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return "", fmt.Errorf("GitHub API rejected request (status %d): %s", resp.StatusCode, msg)
		}
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, "_linux_arm64.tar.gz") {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no _linux_arm64.tar.gz release asset found")
}

func extractFilesFromTarGz(tarGzData []byte, targetFiles []string) (map[string][]byte, error) {
	reader := bytes.NewReader(tarGzData)

	format, stream, err := archives.Identify(context.Background(), "", reader)
	if err != nil {
		return nil, err
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return nil, fmt.Errorf("format does not support extraction")
	}

	extractedFiles := make(map[string][]byte)

	err = extractor.Extract(context.Background(), stream, func(ctx context.Context, fileInfo archives.FileInfo) error {
		baseName := path.Base(fileInfo.Name())
		for _, target := range targetFiles {
			if baseName == target {
				file, err := fileInfo.Open()
				if err != nil {
					return fmt.Errorf("failed to open %s: %w", fileInfo.Name(), err)
				}
				defer func() { _ = file.Close() }()

				content, err := io.ReadAll(file)
				if err != nil {
					return fmt.Errorf("failed to read %s: %w", fileInfo.Name(), err)
				}

				extractedFiles[baseName] = content
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("files not found in archive")
	}

	return extractedFiles, nil
}

func fetchFile(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	addGitHubAuthHeader(req)

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

func FetchLatestReleaseBinaries(githubRepoSlug string, binaries []string) (map[string][]byte, error) {
	url, err := getLatestReleaseTarAddress(githubRepoSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve latest release download URL: %w", err)
	}

	tarball, err := fetchFile(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download latest release: %w", err)
	}

	files, err := extractFilesFromTarGz(tarball, binaries)
	if err != nil {
		return nil, fmt.Errorf("failed to extract files from tar.gz: %w", err)
	}
	return files, err
}

func install(installPath string, targetHost ssh.Host, binaries map[string][]byte) error {
	mode := "0755"

	for binaryName, binaryData := range binaries {
		if err := ssh.ValidateBinaryName(binaryName); err != nil {
			return err
		}

		installCmd := fmt.Sprintf("install -D -m %s /dev/stdin %s/%s", mode, installPath, binaryName)
		_, stderr, err := ssh.Exec(targetHost, installCmd, binaryData)
		if err != nil {
			stderrStr := strings.ToLower(stderr)
			if strings.Contains(stderrStr, "publickey") ||
				strings.Contains(stderrStr, "authentication") ||
				strings.Contains(stderrStr, "connection refused") {
				return fmt.Errorf("%w: %s", ErrSSHAuthFailed, stderr)
			}
			if strings.Contains(stderrStr, "permission denied") ||
				strings.Contains(stderrStr, "cannot create") ||
				strings.Contains(stderrStr, "read-only") {
				return fmt.Errorf("%w: %s", ErrPermissionDenied, stderr)
			}
			return errors.New(stderr)
		}
	}
	return nil
}

// installToFirstWriteableDir attempts to install binaries to the highest preference path that the user has permissions for.
// Silently ignores permission failures until the last path.
// Returns the installation location and a list of installed binary names.
func installToFirstWriteableDir(paths []PathCandidate, targetHost ssh.Host, binaries map[string][]byte) (PathCandidate, []string, error) {
	var binaryNames []string
	for name := range binaries {
		binaryNames = append(binaryNames, name)
	}

	for i, dir := range paths {
		err := install(dir.Path, targetHost, binaries)
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

func InstallBinariesFromGithubRelease(targetHost ssh.Host, repoURL string, binaryNames []string) ([]InstallResult, error) {
	for _, binaryName := range binaryNames {
		if err := ssh.ValidateBinaryName(binaryName); err != nil {
			return nil, err
		}
	}

	binaries, err := FetchLatestReleaseBinaries(repoURL, binaryNames)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release binaries: %w", err)
	}

	existingBinaryPaths := make(map[string]string)
	var binariesNotOnPath []string

	for _, binaryName := range binaryNames {
		existingPath, err := getExistingBinaryDir(targetHost, binaryName)
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
		err := install(dirPath, targetHost, singleBinary)
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
		paths, err := FindPathDirs(targetHost)
		if err != nil {
			return nil, fmt.Errorf("failed to find valid PATH directories: %w", err)
		}

		newBinariesMap := make(map[string][]byte)
		for _, binaryName := range binariesNotOnPath {
			newBinariesMap[binaryName] = binaries[binaryName]
		}

		installLoc, installedBinaries, err := installToFirstWriteableDir(paths, targetHost, newBinariesMap)
		if err != nil {
			return nil, fmt.Errorf("installation of new binaries failed: %w", err)
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
