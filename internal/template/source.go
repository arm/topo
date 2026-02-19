package template

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arm/topo/internal/catalog"
)

type DestDirExistsError struct {
	Dir string
}

func (e DestDirExistsError) Error() string {
	return fmt.Sprintf("directory %s already exists", e.Dir)
}

type Source interface {
	CopyTo(destDir string) error
	String() string
	GetName() (string, error)
}

func NewSource(source string) (Source, error) {
	if strings.HasPrefix(source, "template:") || strings.HasPrefix(source, "git:") || strings.HasPrefix(source, "dir:") {
		parts := strings.SplitN(source, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid source format: %s (expected format: <type>:<value>, e.g., template:hello-world or git:https://github.com/user/repo.git)", source)
		}

		sourceType := parts[0]
		sourceValue := parts[1]

		if sourceValue == "" {
			return nil, fmt.Errorf("source value cannot be empty")
		}

		switch sourceType {
		case "template":
			return TemplateIdSource(sourceValue), nil
		case "git":
			return NewGitSource(sourceValue), nil
		case "dir":
			return DirSource{Path: sourceValue}, nil
		default:
			return nil, fmt.Errorf("unsupported source type: %s (supported: template:, git:, dir:)", sourceType)
		}
	}

	if isGitURL(source) {
		return NewGitSource(source), nil
	}

	sourceType, _, hasType := strings.Cut(source, ":")
	if hasType {
		return nil, fmt.Errorf("unsupported source type: %s (supported: template:, git:, dir:)", sourceType)
	}

	return nil, fmt.Errorf("invalid source format: %s (expected format: <type>:<value>, e.g., template:hello-world or git:https://github.com/user/repo.git)", source)
}

func isGitURL(source string) bool {
	return strings.HasPrefix(source, "git@") ||
		strings.HasPrefix(source, "ssh://") ||
		strings.HasPrefix(source, "https://") ||
		strings.HasPrefix(source, "http://") ||
		strings.HasPrefix(source, "git://")
}

type TemplateIdSource string

func (t TemplateIdSource) CopyTo(destDir string) error {
	templateRepo, err := catalog.GetTemplateRepo(string(t))
	if err != nil {
		return err
	}
	gitSource := GitSource{
		URL: templateRepo.Url,
		Ref: templateRepo.Ref,
	}
	return gitSource.CopyTo(destDir)
}

func (t TemplateIdSource) String() string {
	return fmt.Sprintf("template:%s", string(t))
}

func (t TemplateIdSource) GetName() (string, error) {
	templateRepo, err := catalog.GetTemplateRepo(string(t))
	if err != nil {
		return "", err
	}
	gitSource := GitSource{
		URL: templateRepo.Url,
		Ref: templateRepo.Ref,
	}
	return gitSource.GetName()
}

type GitSource struct {
	URL string
	Ref string
}

func NewGitSource(url string) GitSource {
	if idx := strings.LastIndex(url, "#"); idx != -1 {
		return GitSource{
			URL: url[:idx],
			Ref: url[idx+1:],
		}
	}

	return GitSource{URL: url, Ref: ""}
}

func (g GitSource) CopyTo(destDir string) error {
	if _, err := os.Stat(destDir); err == nil {
		return DestDirExistsError{Dir: destDir}
	}

	args := []string{"clone", "--depth", "1"}
	if g.Ref != "" {
		args = append(args, "--branch", g.Ref)
	}
	args = append(args, g.URL, destDir)
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (g GitSource) String() string {
	if g.Ref != "" {
		return fmt.Sprintf("git:%s#%s", g.URL, g.Ref)
	}
	return fmt.Sprintf("git:%s", g.URL)
}

func (g GitSource) GetName() (string, error) {
	base := filepath.Base(strings.TrimSpace(g.URL))
	base = strings.TrimSuffix(base, ".git")
	return base, nil
}

type DirSource struct {
	Path string
}

func (d DirSource) CopyTo(destDir string) error {
	if _, err := os.Stat(destDir); err == nil {
		return DestDirExistsError{Dir: destDir}
	}

	dstAbs, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}

	srcAbs, err := filepath.Abs(d.Path)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
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

func (d DirSource) String() string {
	return fmt.Sprintf("dir:%s", d.Path)
}

func (d DirSource) GetName() (string, error) {
	return filepath.Base(d.Path), nil
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

func copyFile(src, dst string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close() //nolint:errcheck

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if closeError := dstFile.Close(); closeError != nil && err == nil {
			err = closeError
		}
	}()

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
