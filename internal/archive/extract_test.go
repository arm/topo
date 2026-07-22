package archive_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"testing"

	archiveutil "github.com/arm/topo/internal/archive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFiles(t *testing.T) {
	t.Run("returns requested files by base name", func(t *testing.T) {
		archiveData := createTarGz(t, map[string]string{
			"bin/first":  "first contents",
			"bin/second": "second contents",
			"ignored":    "ignored contents",
		})

		got, err := archiveutil.ExtractFiles(context.Background(), archiveData, []string{"first", "second"})

		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"first":  []byte("first contents"),
			"second": []byte("second contents"),
		}, got)
	})

	t.Run("returns an error when a requested file is absent", func(t *testing.T) {
		archiveData := createTarGz(t, map[string]string{"first": "contents"})

		_, err := archiveutil.ExtractFiles(context.Background(), archiveData, []string{"missing"})

		assert.ErrorContains(t, err, `"missing" not found in archive`)
	})
}

func createTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, contents := range files {
		err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(contents))})
		require.NoError(t, err)
		_, err = tarWriter.Write([]byte(contents))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return output.Bytes()
}
