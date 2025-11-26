package vscode

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/arm-debug/topo-cli/internal/project"
)

func PrintProject(w io.Writer, targetProjectFile string) error {
	project, err := project.Read(targetProjectFile)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}
