package source

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Dir struct {
	Path string
}

func (d Dir) CopyTo(destDir string) error {
	srcAbs, err := filepath.Abs(d.Path)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}

	dstAbs, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}

	if isNestedPath(srcAbs, dstAbs) {
		return fmt.Errorf("destination directory %s is inside source directory %s", dstAbs, srcAbs)
	}

	srcInfo, err := os.Stat(d.Path)
	if err != nil {
		return fmt.Errorf("failed to access source directory: %w", err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", d.Path)
	}

	return copyDir(d.Path, destDir)
}

func (d Dir) String() string {
	return fmt.Sprintf("dir:%s", d.Path)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.Type()&os.ModeSymlink != 0 {
			if err := copySymlink(srcPath, dstPath); err != nil {
				return err
			}
		} else if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(target, dst)
}

func isNestedPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
