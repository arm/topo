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

func createDstFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), upgrade.BinaryName("topo"))
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
		dst := createDstFile(t)

		err := upgrade.Install(context.Background(), dst, srv.URL)

		require.NoError(t, err)
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, fakeContent, string(got))
	})

	t.Run("sets executable permissions on installed binary", func(t *testing.T) {
		testutil.RequireOS(t, "linux", "darwin")
		srv := serveBytes(t, validArchive)
		dst := createDstFile(t)

		err := upgrade.Install(context.Background(), dst, srv.URL)

		require.NoError(t, err)
		info, err := os.Stat(dst)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("leaves no temp files in destination directory", func(t *testing.T) {
		srv := serveBytes(t, validArchive)
		dst := createDstFile(t)

		err := upgrade.Install(context.Background(), dst, srv.URL)

		require.NoError(t, err)
		entries, err := os.ReadDir(filepath.Dir(dst))
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})

	t.Run("returns error on non-200 response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		err := upgrade.Install(context.Background(), createDstFile(t), srv.URL)

		assert.ErrorContains(t, err, "HTTP 404")
	})

	t.Run("returns error on connection failure", func(t *testing.T) {
		err := upgrade.Install(context.Background(), createDstFile(t), "something-invalid")

		assert.Error(t, err)
	})

	t.Run("returns error when binary is absent from archive", func(t *testing.T) {
		archive := makeTarGz(t, map[string][]byte{
			"some-other-file": []byte("not a topo binary"),
		})
		srv := serveBytes(t, archive)

		err := upgrade.Install(context.Background(), createDstFile(t), srv.URL)

		assert.Error(t, err)
	})

	t.Run("returns error on invalid archive body", func(t *testing.T) {
		srv := serveBytes(t, []byte("this is not a valid archive"))

		err := upgrade.Install(context.Background(), createDstFile(t), srv.URL)

		assert.Error(t, err)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		srv := serveBytes(t, validArchive)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := upgrade.Install(ctx, createDstFile(t), srv.URL)

		assert.Error(t, err)
	})
}

func TestArtifactoryDownloadURL(t *testing.T) {
	t.Run("contains the formatted target version", func(t *testing.T) {
		url := upgrade.ArtifactoryDownloadURL("3.13.37")

		assert.Contains(t, url, "v3.13.37")
	})

	t.Run("contains current architecture", func(t *testing.T) {
		url := upgrade.ArtifactoryDownloadURL("3.13.37")

		assert.Contains(t, url, runtime.GOARCH)
	})

	t.Run("maps darwin to macos", func(t *testing.T) {
		testutil.RequireOS(t, "darwin")

		url := upgrade.ArtifactoryDownloadURL("3.13.37")

		assert.Contains(t, url, "/macos/")
	})

	t.Run("uses zip extension on windows", func(t *testing.T) {
		testutil.RequireOS(t, "windows")

		url := upgrade.ArtifactoryDownloadURL("3.13.37")

		assert.Contains(t, url, ".zip")
	})

	t.Run("uses tar.gz extension on linux and darwin", func(t *testing.T) {
		testutil.RequireOS(t, "linux", "darwin")

		url := upgrade.ArtifactoryDownloadURL("3.13.37")

		assert.Contains(t, url, ".tar.gz")
	})
}
