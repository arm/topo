package arguments

import (
	"encoding/json"
	"fmt"
)

type Arg struct {
	Name          string
	Description   string
	Required      bool
	Example       string
	CurrentValues []string
}

type ResolvedArg struct {
	Name  string
	Value string
}

type Provider interface {
	Provide(args []Arg) ([]ResolvedArg, error)
}

func formatCurrentValues(values []string) string {
	formatted, err := json.Marshal(values)
	if err != nil {
		return fmt.Sprintf("%q", values)
	}
	return string(formatted)
}
