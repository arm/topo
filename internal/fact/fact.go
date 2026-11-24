package fact

import (
	"context"
	"fmt"
	"strconv"

	"github.com/arm/topo/internal/runner"
)

var BuiltinTargetProviders = map[string]Provider{
	"builtin.target.memory":     MemoryProvider{},
	"builtin.target.remoteproc": RemoteprocProvider{},
}

type Provider interface {
	Facts(ctx context.Context, targetRunner runner.CommandRunner) (Report, error)
}

type Report map[string]Fact

func (r Report) Validate() error {
	for key, fact := range r {
		if key == "" {
			return fmt.Errorf("invalid fact key: %q", key)
		}
		if err := fact.Value.Validate(); err != nil {
			return fmt.Errorf("invalid fact %q: %w", key, err)
		}
	}
	return nil
}

type Fact struct {
	Value       Value
	Label       string
	Description string
}

type Value struct {
	Type    string
	Boolean bool
	Integer int64
	Text    string
}

func (v Value) String() string {
	switch v.Type {
	case "boolean":
		return strconv.FormatBool(v.Boolean)
	case "integer":
		return strconv.FormatInt(v.Integer, 10)
	case "string":
		return v.Text
	}

	panic(fmt.Sprintf("unsupported value type: %q", v.Type))
}

func (v Value) Validate() error {
	switch v.Type {
	case "boolean":
		if v.Integer == 0 && v.Text == "" {
			return nil
		}
	case "integer":
		if !v.Boolean && v.Text == "" {
			return nil
		}
	case "string":
		if !v.Boolean && v.Integer == 0 {
			return nil
		}
	}
	return fmt.Errorf("invalid fact value type or fields: %q", v.Type)
}
