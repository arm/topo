package core

import (
	"encoding/json"
	"fmt"

	"github.com/arm-debug/topo-cli/configs"
)

// Configuration metadata (embedded config-metadata.json)
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

// ReadConfigMetadata loads embedded metadata JSON.
func ReadConfigMetadata() (ConfigMetadata, error) {
	var info ConfigMetadata
	if err := json.Unmarshal(configs.ConfigMetadataJSON, &info); err != nil {
		return info, fmt.Errorf("failed to unmarshal config metadata: %v", err)
	}
	return info, nil
}

// GetConfigMetadata prints metadata as JSON.
func GetConfigMetadata() error {
	info, err := ReadConfigMetadata()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config metadata: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
