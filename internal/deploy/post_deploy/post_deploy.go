package post_deploy

import (
	"fmt"
	"io"
	"os"

	cmdtext "github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/project"
)

func DefaultMessage(composeFile string) string {
	if composeFile == compose.DefaultFileName() {
		return "Run `topo ps` to see deployed containers"
	}

	return fmt.Sprintf("Run `topo ps -f %s` to see deployed containers", cmdtext.QuoteArg(composeFile))
}

func getSuccessMessage(composeFile string) (string, error) {
	f, err := os.Open(composeFile)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	p, err := project.FromContent(f)
	if err != nil {
		return "", err
	}
	return p.Metadata.DeploymentSuccessMessage, nil
}

func PrintDeploySuccess(output io.Writer, composeFile, defaultMessage string) error {
	successMessage, err := getSuccessMessage(composeFile)
	if err != nil {
		return err
	}
	if successMessage == "" {
		successMessage = defaultMessage
	}

	_, err = fmt.Fprintln(output, successMessage)
	return err
}
