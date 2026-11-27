package post_deploy

import (
	"fmt"
	"io"

	"github.com/arm/topo/internal/project"
)

func getSuccessMessage(preparedProject project.PreparedProject) (string, error) {
	composeProject, err := project.Read(preparedProject)
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

func PrintDeploySuccess(output io.Writer, preparedProject project.PreparedProject, defaultMessage string) error {
	successMessage, err := getSuccessMessage(preparedProject)
	if err != nil {
		return err
	}
	if successMessage == "" {
		successMessage = defaultMessage
	}

	_, err = fmt.Fprintln(output, successMessage)
	return err
}
