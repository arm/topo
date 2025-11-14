package arguments

import (
	"github.com/arm-debug/topo-cli/internal/service"
)

type Provider interface {
	Provide(specs []service.ArgSpec) (map[string]string, error)
	Name() string
}
