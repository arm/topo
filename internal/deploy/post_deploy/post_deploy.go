package post_deploy

import (
	"fmt"
	"io"

	"github.com/arm/topo/internal/project"
)

func getSuccessMessage(scope project.Scope) (string, error) {
	composeProject, err := project.Read(scope)
	if err != nil {
		return "", err
	}

	var metadata struct {
		DeploymentSuccessMessage string `mapstructure:"deployment_success_message"`
	}
	found, err := composeProject.Extensions.Get("x-topo", &metadata)
	if err != nil || !found {
		return "", err
	}
	return metadata.DeploymentSuccessMessage, nil
}

func PrintDeploySuccess(output io.Writer, scope project.Scope, defaultMessage string) error {
	successMessage, err := getSuccessMessage(scope)
	if err != nil {
		return err
	}
	if successMessage == "" {
		successMessage = defaultMessage
	}

	_, err = fmt.Fprintln(output, successMessage)
	return err
}
