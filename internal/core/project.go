package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arm-debug/topo-cli/configs"
	"github.com/arm-debug/topo-cli/internal/core/compose"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

type CloneFunc func(url, dest, ref string) error
type GetTemplateFn func(id string) (*template.ServiceTemplateRepo, error)

// Expose core.cloneProject via a wrapper because it is unexported.
func CloneProject(url, dest, ref string) error {
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, url, dest)
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ReadProject parses compose file into a compose-go project.
func ReadProject(targetProjectFile string) (*types.Project, error) {
	ctx := context.Background()
	options, err := cli.NewProjectOptions([]string{targetProjectFile}, cli.WithOsEnv, cli.WithDotEnv, cli.WithResolvedPaths(false), cli.WithNormalization(false))
	if err != nil {
		return nil, err
	}
	return cli.ProjectFromOptions(ctx, options)
}

// GetProject prints the compose project JSON.
func GetProject(targetProjectFile string) error {
	project, err := ReadProject(targetProjectFile)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func RunAddService(targetProjectFile, gitURL, gitRef, newServiceName string, cloner CloneFunc) error {
	project, err := ReadProject(targetProjectFile)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}

	destDir := filepath.Join(filepath.Dir(targetProjectFile), newServiceName)

	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("directory %s already exists; please choose a different service name or remove the existing directory", destDir)
	}

	if err := cloner(gitURL, destDir, gitRef); err != nil {
		return fmt.Errorf("failed to clone Service Template: %w", err)
	}

	serviceManifest, err := template.ParseServiceDefinition(destDir)
	if err != nil {
		return fmt.Errorf("failed to load topo service from repo %s: %w", gitURL, err)
	}

	newSvc, err := compose.ParseServiceFromTopo(newServiceName, &serviceManifest)
	if err != nil {
		return err
	}

	if err := compose.InsertService(project, newSvc); err != nil {
		return err
	}

	compose.RegisterNamedVolumes(project, newSvc)

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(project); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(targetProjectFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", targetProjectFile, err)
	}
	return nil
}

func RunAddServiceByTemplateId(targetProjectFile, templateId, newServiceName string, cloner CloneFunc, getTemplate GetTemplateFn) error {
	serviceTemplateRepo, err := getTemplate(templateId)
	if err != nil {
		return err
	}
	return RunAddService(targetProjectFile, serviceTemplateRepo.Url, serviceTemplateRepo.Ref, newServiceName, cloner)
}

// RunRemoveService deletes a service entry.
func RunRemoveService(composeFilePath, serviceName string) error {
	project, err := ReadProject(composeFilePath)
	if err != nil {
		return err
	}
	newServices := types.Services{}
	for k, svc := range project.Services {
		if k == serviceName {
			continue
		}
		newServices[k] = svc
	}
	project.Services = newServices
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(project); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(composeFilePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", composeFilePath, err)
	}
	return nil
}

func RunInitProject(projectDir string, sshTarget string) error {
	composePath := filepath.Join(projectDir, DefaultComposeFileName)
	if _, err := os.Stat(composePath); err == nil {
		return fmt.Errorf("compose file already exists at %s", composePath)
	} else if !os.IsNotExist(err) {
		return err
	}
	compose := types.Project{
		Services: types.Services{},
	}
	data, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal compose file: %w", err)
	}
	if err := os.WriteFile(composePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}
	return GenerateMakefile(composePath, sshTarget)
}

// GenerateMakefile materializes a Makefile from template adjusting compose filename & ssh target.
func GenerateMakefile(composePath string, sshTarget string) error {
	dockerContext := getContextName(sshTarget)
	composeFilename := filepath.Base(composePath)
	dir := filepath.Dir(composePath)
	template := string(configs.MakefileTemplate)
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "COMPOSE_FILE"):
			lines[i] = fmt.Sprintf("COMPOSE_FILE    ?= %s", composeFilename)
		case strings.HasPrefix(line, "SSH_TARGET "):
			lines[i] = fmt.Sprintf("SSH_TARGET      := %s", sshTarget)
		case strings.HasPrefix(line, "DOCKER_CONTEXT "):
			lines[i] = fmt.Sprintf("DOCKER_CONTEXT  := %s", dockerContext)
		}
	}
	result := strings.Join(lines, "\n")
	return os.WriteFile(filepath.Join(dir, "Makefile"), []byte(result), 0644)
}
