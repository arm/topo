package project

import "github.com/arm/topo/internal/env"

type Scope struct {
	ComposeFile string
	Env         []string
}

func LoadScope(composeFile string, target string) (Scope, error) {
	targetEnv, err := env.ResolveTargetEnv(target)
	if err != nil {
		return Scope{}, err
	}

	return Scope{
		ComposeFile: composeFile,
		Env:         targetEnv,
	}, nil
}
