package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var catalogSchemaVersionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

const (
	quicktypeVersion     = "26.0.0"
	defaultSchemaBaseURL = "https://artifacts.tools.arm.com/devx-topo-project-catalog/"
	generationTimeout    = 2 * time.Minute
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

	ctx, cancel := context.WithTimeout(context.Background(), generationTimeout)
	defer cancel()

	outputFile, err := generatedOutputPath()
	if err != nil {
		return err
	}
	return generateTypes(ctx, schemaURL, catalogSchemaVersion, outputFile)
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

func generateTypes(ctx context.Context, schemaURL string, catalogVersion string, outputFile string) error {
	// #nosec G702 -- schemaURL is passed as an argument without invoking a shell.
	command := exec.CommandContext(ctx, "npx", "--yes", "quicktype@"+quicktypeVersion,
		"--src-lang", "schema", "--lang", "go", "--package", "catalog",
		"--top-level", "CatalogDocument", schemaURL)
	command.Stderr = os.Stderr
	generated, err := command.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("quicktype timed out after %s while reading schema %q", generationTimeout, schemaURL)
		}
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("starting quicktype failed: npx was not found; install Node.js 20 or newer: %w", err)
		}
		return fmt.Errorf("quicktype %s failed for schema %q: %w", quicktypeVersion, schemaURL, err)
	}

	formatted, err := addCatalogVersionConstant(generated, catalogVersion)
	if err != nil {
		return fmt.Errorf("failed to format quicktype output from schema %q: %w", schemaURL, err)
	}
	// #nosec G703 -- outputFile is derived from the generator source location.
	if err := os.WriteFile(outputFile, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write generated types to %q: %w", outputFile, err)
	}
	return nil
}

func addCatalogVersionConstant(generated []byte, version string) ([]byte, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "catalog_schema_generated.go", generated, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quicktype output: %w", err)
	}

	versionDeclaration := &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent("CatalogMajorVersion")},
			Values: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(majorVersion(version))}},
		}},
	}
	declarationIndex := 0
	for index, declaration := range file.Decls {
		if generalDeclaration, ok := declaration.(*ast.GenDecl); ok && generalDeclaration.Tok == token.IMPORT {
			declarationIndex = index + 1
		}
	}
	file.Decls = append(file.Decls, nil)
	copy(file.Decls[declarationIndex+1:], file.Decls[declarationIndex:])
	file.Decls[declarationIndex] = versionDeclaration

	var formatted bytes.Buffer
	if err := format.Node(&formatted, fileSet, file); err != nil {
		return nil, fmt.Errorf("failed to format generated types: %w", err)
	}
	return formatted.Bytes(), nil
}

func majorVersion(version string) string {
	major, _, _ := strings.Cut(version, ".")
	return major
}
