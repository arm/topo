package project

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"github.com/compose-spec/compose-go/v2/template"
)

func Configure(scope Scope, resolver parameter.Resolver) error {
	currentValues, err := env.CurrentValues(scope.EnvFiles)
	if err != nil {
		return fmt.Errorf("failed to load current environment values: %w", err)
	}

	definitions, err := LoadParameterDefinitions(scope.ComposeFile)
	if err != nil {
		return fmt.Errorf("failed to load parameter definitions: %w", err)
	}

	if len(definitions) == 0 {
		return nil
	}

	if err := warnUnreferencedParameters(scope.ComposeFile, definitions); err != nil {
		return err
	}

	values, err := resolver.Resolve(definitions, currentValues)
	if err != nil {
		return fmt.Errorf("failed to collect parameter values: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	root := filepath.Dir(scope.ComposeFile)
	outputFiles, err := env.ResolveFiles(root, []string{env.DefaultFilename}, true)
	if err != nil {
		return fmt.Errorf("failed to resolve output environment file: %w", err)
	}
	outputValues, err := env.ReadFiles(outputFiles)
	if err != nil {
		return fmt.Errorf("failed to load output environment values: %w", err)
	}
	maps.Copy(outputValues, values)
	return env.WriteFile(filepath.Join(root, env.DefaultFilename), outputValues)
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
