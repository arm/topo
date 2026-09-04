package parameter

import (
	"encoding/json"
	"fmt"
)

type Definition struct {
	Name          string
	Description   string
	Required      bool
	Example       string
	CurrentValues []string
}

type Values map[string]string

type Provider interface {
	Provide(definitions []Definition) (Values, error)
}

func formatCurrentValues(values []string) string {
	formatted, err := json.Marshal(values)
	if err != nil {
		return fmt.Sprintf("%q", values)
	}
	return string(formatted)
}
