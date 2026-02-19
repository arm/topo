package vscode

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/arm/topo/internal/compose"
)

func PrintProject(w io.Writer, targetProjectFile string) error {
	project, err := compose.ReadProject(targetProjectFile)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}
