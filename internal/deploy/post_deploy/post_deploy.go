package post_deploy

import (
	"fmt"
	"io"
	"os"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/template"
)

type DeploySuccess struct {
	composeFile    string
	host           command.Host
	defaultMessage string
}

func NewDeploySuccess(composeFile string, h command.Host, defaultMessage string) *DeploySuccess {
	return &DeploySuccess{
		composeFile:    composeFile,
		host:           h,
		defaultMessage: defaultMessage,
	}
}

func (t *DeploySuccess) Description() string {
	return "Deployment Success"
}

func getSuccessMessage(composeFile string) (string, error) {
	f, err := os.Open(composeFile)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	tpl, err := template.FromContent(f)
	if err != nil {
		return "", err
	}
	return tpl.Metadata.DeploymentSuccessMessage, nil
}

func (p *DeploySuccess) Run(w io.Writer) error {
	successMessage, err := getSuccessMessage(p.composeFile)
	if err != nil {
		return err
	}
	if successMessage == "" {
		successMessage = p.defaultMessage
	}

	_, err = fmt.Fprintln(w, successMessage)
	if err != nil {
		return err
	}

	return nil
}
