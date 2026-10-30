package project

import (
	"github.com/arm/topo/internal/env"
)

type Scope struct {
	ComposeFiles []string
	Env          []string
	EnvFiles     []string
}

func BuildScope(composeFiles []string, target string, envFiles []string) (Scope, error) {
	targetEnv, err := env.ResolveTargetEnv(target)
	if err != nil {
		return Scope{}, err
	}

	return Scope{
		ComposeFiles: composeFiles,
		Env:          targetEnv,
		EnvFiles:     envFiles,
	}, nil
}
