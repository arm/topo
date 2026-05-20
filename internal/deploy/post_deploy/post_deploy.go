package post_deploy

import (
	"fmt"
	"io"
	"os"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/template"
)

type PostDeployMessage struct {
	composeFile string
	host        command.Host
}

func NewPostDeployMessage(composeFile string, h command.Host) *PostDeployMessage {
	return &PostDeployMessage{
		composeFile: composeFile,
		host:        h,
	}
}

func (t *PostDeployMessage) Description() string {
	return "Deploy Success"
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
	return tpl.Metadata.SuccessMessage, nil
}

func (p *PostDeployMessage) Run(w io.Writer) error {
	if w == nil {
		return nil
	}
	successMessage, err := getSuccessMessage(p.composeFile)
	if err != nil {
		return err
	}
	if successMessage == "" {
		successMessage = "Run `topo ps` to see deployed containers"
	}

	_, err = fmt.Fprintln(w, successMessage)
	if err != nil {
		return err
	}

	return nil
}
