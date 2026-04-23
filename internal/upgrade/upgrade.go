package upgrade

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/arm/topo/internal/version"
	"github.com/mholt/archives"
)

var topoBinaryName = binaryName("topo")

func Install(ctx context.Context, binPath string, downloadURL string) error {
	archivePath, err := downloadArchive(ctx, downloadURL)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath) //nolint:errcheck

	newBinaryPath, err := extractBinary(ctx, archivePath)
	if err != nil {
		return err
	}
	defer os.Remove(newBinaryPath) //nolint:errcheck

	return moveBinary(newBinaryPath, binPath)
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

func downloadArchive(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}
	// #nosec G107 -- URL is constructed from a hardcoded, trusted base URL
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download archive from %s: HTTP %d", url, resp.StatusCode)
	}

	archive, err := os.CreateTemp("", "topo-archive-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	archivePath := archive.Name()

	if _, err := io.Copy(archive, resp.Body); err != nil {
		_ = archive.Close()
		_ = os.Remove(archivePath)
		return "", fmt.Errorf("saving archive: %w", err)
	}
	if err := archive.Close(); err != nil {
		_ = os.Remove(archivePath)
		return "", fmt.Errorf("flushing archive: %w", err)
	}

	return archivePath, nil
}

func extractBinary(ctx context.Context, archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close() //nolint:errcheck

	format, stream, err := archives.Identify(ctx, archivePath, f)
	if err != nil {
		return "", fmt.Errorf("failed to identify archive format: %w", err)
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return "", fmt.Errorf("archive format does not support extraction")
	}

	var extractedPath string

	err = extractor.Extract(ctx, stream, func(ctx context.Context, fileInfo archives.FileInfo) error {
		if filepath.Base(fileInfo.Name()) != topoBinaryName {
			return nil
		}

		rc, err := fileInfo.Open()
		if err != nil {
			return fmt.Errorf("failed to open %s in archive: %w", fileInfo.Name(), err)
		}
		defer rc.Close() //nolint:errcheck

		tmp, err := os.CreateTemp("", binaryName("topo-new-*"))
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		extractedPath = tmp.Name()

		// #nosec G110 -- archive from a hardcoded, trusted Artifactory URL
		if _, err := io.Copy(tmp, rc); err != nil {
			_ = tmp.Close()
			_ = os.Remove(extractedPath)
			extractedPath = ""
			return fmt.Errorf("failed to extract binary: %w", err)
		}
		return tmp.Close()
	})
	if err != nil {
		if extractedPath != "" {
			_ = os.Remove(extractedPath)
		}
		return "", fmt.Errorf("failed to read archive: %w", err)
	}

	if extractedPath == "" {
		return "", fmt.Errorf("failed to find %q in archive", topoBinaryName)
	}

	return extractedPath, nil
}

func moveBinary(srcPath string, dstPath string) error {
	if runtime.GOOS == "windows" {
		oldPath := dstPath + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(dstPath, oldPath); err != nil {
			return fmt.Errorf("failed to rename current binary: %w", err)
		}
		if err := os.Rename(srcPath, dstPath); err != nil {
			_ = os.Rename(oldPath, dstPath) // attempt to restore
			return fmt.Errorf("failed to move new binary: %w", err)
		}
		_ = os.Remove(oldPath)
		return nil
	}

	err := os.Rename(srcPath, dstPath)
	if err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	err = os.Chmod(dstPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

func binaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
