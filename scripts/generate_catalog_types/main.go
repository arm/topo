package main

import (
	"errors"
	"fmt"
	"go/format"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var catalogSchemaVersionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

const (
	quicktypeVersion     = "26.0.0"
	defaultSchemaBaseURL = "https://artifacts.tools.arm.com/devx-topo-project-catalog/"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "generating catalog types failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || os.Args[1] == "" {
		return fmt.Errorf("expected one catalog version; usage: go run ./scripts/generate_catalog_types VERSION")
	}

	catalogSchemaVersion := os.Args[1]
	if err := validateCatalogVersion(catalogSchemaVersion); err != nil {
		return err
	}
	schemaURL := defaultSchemaBaseURL + url.PathEscape(catalogSchemaVersion) + "/catalog/catalog.schema.json"

	outputFile, err := generatedOutputPath()
	if err != nil {
		return err
	}
	return generateTypes(schemaURL, catalogSchemaVersion, outputFile)
}

func validateCatalogVersion(version string) error {
	if !catalogSchemaVersionPattern.MatchString(version) {
		return fmt.Errorf("catalog version %q must use vMAJOR.MINOR.PATCH format, for example v1.1.2", version)
	}
	return nil
}

func generatedOutputPath() (string, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("determining repository root: generator source location is unavailable")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	return filepath.Join(repositoryRoot, "internal", "catalog", "catalog_schema_generated.go"), nil
}

func generateTypes(schemaURL string, catalogVersion string, outputFile string) error {
	// #nosec G702 -- schemaURL is passed as an argument without invoking a shell.
	command := exec.Command("npx", "--yes", "quicktype@"+quicktypeVersion,
		"--src-lang", "schema", "--lang", "go", "--package", "catalog",
		"--top-level", "CatalogDocument", schemaURL)
	command.Stderr = os.Stderr
	generated, err := command.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("starting quicktype failed: npx was not found; install Node.js 20 or newer: %w", err)
		}
		return fmt.Errorf("quicktype %s failed for schema %q: %w", quicktypeVersion, schemaURL, err)
	}

	generated = fmt.Appendf(generated, "\nconst CatalogMajorVersion = %q\n", majorVersion(catalogVersion))
	formatted, err := format.Source(generated)
	if err != nil {
		return fmt.Errorf("failed to format quicktype output from schema %q: %w", schemaURL, err)
	}
	// #nosec G703 -- outputFile is derived from the generator source location.
	if err := os.WriteFile(outputFile, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write generated types to %q: %w", outputFile, err)
	}
	return nil
}

func majorVersion(version string) string {
	major, _, _ := strings.Cut(version, ".")
	return major
}
