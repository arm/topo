package source

import (
	"fmt"

	"github.com/arm-debug/topo-cli/internal/service"
)

type TemplateId string

func (t TemplateId) CopyTo(destDir string) error {
	serviceTemplateRepo, err := service.GetTemplateRepo(string(t))
	if err != nil {
		return err
	}
	gitSource := Git{
		URL: serviceTemplateRepo.Url,
		Ref: serviceTemplateRepo.Ref,
	}
	return gitSource.CopyTo(destDir)
}

func (t TemplateId) String() string {
	return fmt.Sprintf("template:%s", string(t))
}
