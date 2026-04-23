package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/version"
	"github.com/mholt/archives"
)

func Install(ctx context.Context, currentBin string, downloadURL string) error {
	archiveData, err := downloadArchive(ctx, downloadURL)
	if err != nil {
		return err
	}

	newBin, err := extractBinary(ctx, archiveData, filepath.Dir(currentBin))
	if err != nil {
		return err
	}
	defer ensureFileRemoved(newBin)

	return moveBinary(newBin, currentBin)
}

func CurrentBinaryPath() (string, error) {
	binPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to determine current executable path: %w", err)
	}
	binPath, err = filepath.EvalSymlinks(binPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	return binPath, nil
}

func ArtifactoryDownloadURL(targetVersion string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	urlOS := runtime.GOOS
	if runtime.GOOS == "darwin" {
		urlOS = "macos"
	}

	archiveName := fmt.Sprintf("topo_%s_%s.%s", runtime.GOOS, runtime.GOARCH, ext)
	return fmt.Sprintf("%s/v%s/%s/%s", version.ArtifactoryBaseURL, targetVersion, urlOS, archiveName)
}

func BinaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func downloadArchive(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	// #nosec G704 -- request to a hardcoded, trusted URL
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download archive from %s: HTTP %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}

	return data, nil
}

func extractBinary(ctx context.Context, archiveData []byte, destDir string) (string, error) {
	format, stream, err := archives.Identify(ctx, "", bytes.NewReader(archiveData))
	if err != nil {
		return "", fmt.Errorf("failed to identify archive format: %w", err)
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return "", fmt.Errorf("archive format does not support extraction")
	}

	var tmpPath string

	err = extractor.Extract(ctx, stream, func(ctx context.Context, fileInfo archives.FileInfo) error {
		if filepath.Base(fileInfo.Name()) != BinaryName("topo") {
			return nil
		}

		rc, err := fileInfo.Open()
		if err != nil {
			return fmt.Errorf("failed to open %s in archive: %w", fileInfo.Name(), err)
		}
		defer rc.Close() //nolint:errcheck

		// important to create the temp file in the same directory to ensure atomic rename works across filesystems
		tmp, err := os.CreateTemp(destDir, BinaryName(".topo-new-*"))
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath = tmp.Name()

		// #nosec G110 -- archive from a hardcoded, trusted Artifactory URL
		if _, err := io.Copy(tmp, rc); err != nil {
			tmp.Close() //nolint:errcheck
			return fmt.Errorf("failed to extract binary: %w", err)
		}
		return tmp.Close()
	})
	if err != nil {
		ensureFileRemoved(tmpPath)
		return "", fmt.Errorf("failed to read archive: %w", err)
	}

	return tmpPath, nil
}

func moveBinary(src, dst string) error {
	if runtime.GOOS == "windows" {
		old := dst + ".old"
		if err := os.Rename(dst, old); err != nil {
			return fmt.Errorf("failed to rename current binary: %w", err)
		}

		if err := os.Rename(src, dst); err != nil {
			restoreErr := os.Rename(old, dst)
			if restoreErr != nil {
				logger.Error(fmt.Sprintf("failed to restore original binary after failed upgrade: %v", restoreErr))
			}
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		return nil
	}

	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	if err := os.Chmod(dst, 0o755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

func ensureFileRemoved(path string) {
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.Warn(fmt.Sprintf("failed to remove temporary file %s: %v", path, err))
	}
}
