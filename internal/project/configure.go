package project

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"github.com/compose-spec/compose-go/v2/template"
)

func Configure(composeFilePath string, resolver parameter.Resolver) error {
	envFile := filepath.Join(filepath.Dir(composeFilePath), env.DefaultFilename)
	currentValues, err := env.ReadFile(envFile)
	if errors.Is(err, os.ErrNotExist) {
		currentValues = make(map[string]string)
	} else if err != nil {
		return fmt.Errorf("failed to load current environment values: %w", err)
	}

	definitions, err := LoadParameterDefinitions(composeFilePath, currentValues)
	if err != nil {
		return fmt.Errorf("failed to load parameter definitions: %w", err)
	}

	if len(definitions) == 0 {
		return nil
	}

	if err := warnUnreferencedParameters(composeFilePath, definitions); err != nil {
		return err
	}

	values, err := resolver.Resolve(definitions)
	if err != nil {
		return fmt.Errorf("failed to collect parameter values: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	maps.Copy(currentValues, values)
	return env.WriteFile(envFile, currentValues)
}

func warnUnreferencedParameters(composeFilePath string, definitions []parameter.Definition) error {
	model, err := readUninterpolated(composeFilePath)
	if err != nil {
		return err
	}
	referencedEnvVars := template.ExtractVariables(model, nil)

	unreferencedDefinitions := slices.DeleteFunc(slices.Clone(definitions), func(d parameter.Definition) bool {
		_, referenced := referencedEnvVars[d.Name]
		return referenced
	})

	for _, param := range unreferencedDefinitions {
		logger.Warn(fmt.Sprintf("parameter %q is not referenced through an environment variable in the Compose file; configuring it will have no effect", param.Name))
	}

	return nil
}
