package vscode

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
)

//go:embed config-metadata.json
var configMetadataJSON []byte

type ConfigMetadata struct {
	Boards []BoardInfo `json:"boards"`
}

type BoardInfo struct {
	ID         string          `json:"id"`
	Name       string          `json:"name,omitempty"`
	Subsystems []SubsystemInfo `json:"subsystems"`
}

type SubsystemInfo struct {
	ID         string            `json:"id"`
	Runtime    string            `json:"runtime"`
	Annotation map[string]string `json:"annotation"`
}

func ReadConfigMetadata() (ConfigMetadata, error) {
	var info ConfigMetadata
	if err := json.Unmarshal(configMetadataJSON, &info); err != nil {
		return info, fmt.Errorf("failed to unmarshal config metadata: %v", err)
	}
	return info, nil
}

func PrintConfigMetadata(w io.Writer) error {
	info, err := ReadConfigMetadata()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config metadata: %w", err)
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}
