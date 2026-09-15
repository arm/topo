package testutil

import (
	"errors"
	"fmt"
	"os"
)

type Flock struct {
	file *os.File
}

func AcquireFlock(path string) (*Flock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := lockFile(file); err != nil {
		return nil, errors.Join(fmt.Errorf("acquire file lock: %w", err), file.Close())
	}
	return &Flock{file: file}, nil
}

func (f *Flock) Release() {
	if err := f.file.Close(); err != nil {
		panic(fmt.Errorf("release file lock: %w", err))
	}
}
