package post_deploy

import (
	"fmt"
	"io"

	cmdtext "github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/ssh"
)

func DefaultMessage(composeFile string) string {
	if composeFile == compose.DefaultFileName() {
		return "Run `topo ps` to see deployed containers"
	}

	return fmt.Sprintf("Run `topo ps -f %s` to see deployed containers", cmdtext.QuoteArg(composeFile))
}

func getSuccessMessage(composeFile string, target ssh.Destination) (string, error) {
	composeProject, err := compose.ReadProjectWithEnvironment(composeFile, env.ComposeEnv(target))
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

func PrintDeploySuccess(output io.Writer, composeFile string, target ssh.Destination, defaultMessage string) error {
	successMessage, err := getSuccessMessage(composeFile, target)
	if err != nil {
		return err
	}
	if successMessage == "" {
		successMessage = defaultMessage
	}

	_, err = fmt.Fprintln(output, successMessage)
	return err
}
