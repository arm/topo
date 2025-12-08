package source

import (
	"fmt"
)

type DestDirExistsError struct {
	Dir string
}

func (e DestDirExistsError) Error() string {
	return fmt.Sprintf("directory %s already exists", e.Dir)
}

type TemplateSource interface {
	CopyTo(destDir string) error
	String() string
}
