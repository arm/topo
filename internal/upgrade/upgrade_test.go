package upgrade_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/arm/topo/internal/upgrade"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))})
		require.NoError(t, err)
		_, err = tw.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	return buf.Bytes()
}

func createFakeBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), upgrade.BinaryName("topo"))
	err := os.WriteFile(path, []byte("old binary"), 0o644)
	require.NoError(t, err)
	return path
}

func serveBytes(t *testing.T, data []byte) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestInstall(t *testing.T) {
	const fakeContent = "fake topo binary"

	validArchive := makeTarGz(t, map[string][]byte{
		upgrade.BinaryName("topo"): []byte(fakeContent),
	})

	t.Run("writes binary to destination", func(t *testing.T) {
		srv := serveBytes(t, validArchive)
		dst := createFakeBinary(t)

		err := upgrade.Install(context.Background(), dst, srv.URL)

		require.NoError(t, err)
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, fakeContent, string(got))
	})

	t.Run("sets executable permissions on installed binary", func(t *testing.T) {
		testutil.RequireOS(t, "linux", "darwin")
		srv := serveBytes(t, validArchive)
		dst := createFakeBinary(t)

		err := upgrade.Install(context.Background(), dst, srv.URL)

		require.NoError(t, err)
		info, err := os.Stat(dst)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("leaves no unexpected temp files in destination directory", func(t *testing.T) {
		srv := serveBytes(t, validArchive)
		dst := createFakeBinary(t)

		err := upgrade.Install(context.Background(), dst, srv.URL)

		require.NoError(t, err)
		entries, err := os.ReadDir(filepath.Dir(dst))
		require.NoError(t, err)

		if runtime.GOOS == "windows" {
			assert.Len(t, entries, 2)
		} else {
			assert.Len(t, entries, 1)
		}
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		err := upgrade.Install(context.Background(), createFakeBinary(t), srv.URL)

		assert.ErrorContains(t, err, "HTTP 404")
	})

	t.Run("returns error on connection failure", func(t *testing.T) {
		err := upgrade.Install(context.Background(), createFakeBinary(t), "something-invalid")

		assert.Error(t, err)
	})

	t.Run("returns error when binary is absent from archive", func(t *testing.T) {
		archive := makeTarGz(t, map[string][]byte{
			"some-other-file": []byte("not a topo binary"),
		})
		srv := serveBytes(t, archive)

		err := upgrade.Install(context.Background(), createFakeBinary(t), srv.URL)

		assert.Error(t, err)
	})

	t.Run("returns error on invalid archive body", func(t *testing.T) {
		srv := serveBytes(t, []byte("this is not a valid archive"))

		err := upgrade.Install(context.Background(), createFakeBinary(t), srv.URL)

		assert.Error(t, err)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		srv := serveBytes(t, validArchive)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := upgrade.Install(ctx, createFakeBinary(t), srv.URL)

		assert.Error(t, err)
	})
}

func TestArtifactoryDownloadURL(t *testing.T) {
	version := "3.13.37"
	tests := []struct {
		os       string
		arch     string
		expected string
	}{
		{"darwin", "amd64", "https://artifacts.tools.arm.com/topo/v" + version + "/macos/topo_darwin_amd64.tar.gz"},
		{"darwin", "arm64", "https://artifacts.tools.arm.com/topo/v" + version + "/macos/topo_darwin_arm64.tar.gz"},
		{"linux", "amd64", "https://artifacts.tools.arm.com/topo/v" + version + "/linux/topo_linux_amd64.tar.gz"},
		{"linux", "arm64", "https://artifacts.tools.arm.com/topo/v" + version + "/linux/topo_linux_arm64.tar.gz"},
		{"windows", "amd64", "https://artifacts.tools.arm.com/topo/v" + version + "/windows/topo_windows_amd64.zip"},
		{"windows", "arm64", "https://artifacts.tools.arm.com/topo/v" + version + "/windows/topo_windows_arm64.zip"},
	}

	for _, tt := range tests {
		name := tt.os + "/" + tt.arch
		t.Run(name, func(t *testing.T) {
			url := upgrade.ArtifactoryDownloadURL(tt.os, tt.arch, version)

			assert.Equal(t, url, tt.expected)
		})
	}
}
