package project

import (
	"github.com/arm/topo/internal/env"
)

type Scope struct {
	ComposeFile string
	Env         []string
	EnvFiles    []string
}

func BuildScope(composeFile string, target string, envFiles []string) (Scope, error) {
	targetEnv, err := env.ResolveTargetEnv(target)
	if err != nil {
		return Scope{}, err
	}

	return Scope{
		ComposeFile: composeFile,
		Env:         targetEnv,
		EnvFiles:    envFiles,
	}, nil
}
