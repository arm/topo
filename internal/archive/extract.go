package archive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/mholt/archives"
)

func ExtractFiles(ctx context.Context, archiveData []byte, fileNames []string) (map[string][]byte, error) {
	requested := make(map[string]struct{}, len(fileNames))
	for _, fileName := range fileNames {
		requested[fileName] = struct{}{}
	}

	format, stream, err := archives.Identify(ctx, "", bytes.NewReader(archiveData))
	if err != nil {
		return nil, fmt.Errorf("identifying archive format: %w", err)
	}

	extractor, ok := format.(archives.Extractor)
	if !ok {
		return nil, fmt.Errorf("archive format does not support extraction")
	}

	extracted := make(map[string][]byte, len(requested))
	err = extractor.Extract(ctx, stream, func(_ context.Context, fileInfo archives.FileInfo) error {
		fileName := filepath.Base(fileInfo.Name())
		if _, ok := requested[fileName]; !ok {
			return nil
		}

		file, err := fileInfo.Open()
		if err != nil {
			return fmt.Errorf("opening %s: %w", fileInfo.Name(), err)
		}

		content, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return fmt.Errorf("reading archive entry %s: %w", fileInfo.Name(), err)
		}
		extracted[fileName] = content
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("extracting archive: %w", err)
	}

	for fileName := range requested {
		if _, ok := extracted[fileName]; !ok {
			return nil, fmt.Errorf("%q not found in archive", fileName)
		}
	}
	return extracted, nil
}
