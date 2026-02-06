package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arm-debug/topo-cli/configs"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

// A compose project
type Project struct {
	Name     string             `yaml:"name" json:"name"`
	Services map[string]Service `yaml:"services" json:"services"`
}

type Service struct {
	Build       *Build            `yaml:"build,omitempty" json:"build,omitempty"`
	Runtime     string            `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Platform    string            `yaml:"platform,omitempty" json:"platform,omitempty"`
	Ports       []ServicePort     `yaml:"ports,omitempty" json:"ports,omitempty"`
}

type Build struct {
	Context string `yaml:"context,omitempty" json:"context,omitempty"`
}

type ServicePort struct {
	Published string `yaml:"published,omitempty" json:"published,omitempty"`
	Target    uint32 `yaml:"target,omitempty" json:"target,omitempty"`
	Protocol  string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

type CloneFunc func(url, dest string) error

// Expose core.cloneProject via a wrapper because it is unexported.
func CloneProject(url, dest string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", url, dest)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ReadProject parses compose file into a compose-go project.
func ReadProject(composeFilePath string) (*types.Project, error) {
	ctx := context.Background()
	options, err := cli.NewProjectOptions([]string{composeFilePath}, cli.WithOsEnv, cli.WithDotEnv, cli.WithResolvedPaths(false), cli.WithNormalization(false))
	if err != nil {
		return nil, err
	}
	return cli.ProjectFromOptions(ctx, options)
}

// GetProject prints the compose project JSON.
func GetProject(composeFilePath string) error {
	project, err := ReadProject(composeFilePath)
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

// RunAddService inserts a template-based service.
func RunAddService(composePath, templateId, newServiceName string, cloner CloneFunc) error {
	templates, err := ReadTemplates()
	if err != nil {
		return err
	}
	var selected *Template
	for i := range templates {
		if templates[i].Id == templateId {
			selected = &templates[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("template with id %q not found", templateId)
	}
	project, err := ReadProject(composePath)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}
	contextPath := "./" + newServiceName
	meta, err := ReadConfigMetadata()
	if err != nil {
		return fmt.Errorf("failed to read config metadata: %w", err)
	}
	var runtime string
	ann := map[string]string{}
	for _, b := range meta.Boards {
		if b.ID == DefaultBoard {
			for _, ss := range b.Subsystems {
				if ss.ID == selected.Subsystem {
					runtime = ss.Runtime
					ann = ss.Annotation
					break
				}
			}
			break
		}
	}
	newSvc := types.ServiceConfig{Name: newServiceName, Build: &types.BuildConfig{Context: contextPath}, Runtime: runtime, Annotations: ann, Platform: selected.Platform}
	if len(selected.Ports) > 0 {
		newSvc.Ports = make([]types.ServicePortConfig, len(selected.Ports))
		for i, p := range selected.Ports {
			parts := strings.Split(p, ":")
			if len(parts) == 2 {
				tgt, err := strconv.ParseUint(parts[1], 10, 32)
				if err != nil {
					return fmt.Errorf("invalid target port %q: %w", parts[1], err)
				}
				newSvc.Ports[i] = types.ServicePortConfig{Published: parts[0], Target: uint32(tgt), Protocol: "tcp"}
			} else {
				tgt, err := strconv.ParseUint(p, 10, 32)
				if err != nil {
					return fmt.Errorf("invalid target port %q: %w", p, err)
				}
				newSvc.Ports[i] = types.ServicePortConfig{Target: uint32(tgt), Protocol: "tcp"}
			}
		}
	}
	if err := insertService(project, newSvc); err != nil {
		return err
	}
	relativisePaths(project)
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(project); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(composePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", composePath, err)
	}
	destDir := filepath.Join(filepath.Dir(composePath), contextPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if err := cloner(selected.Url, destDir); err != nil {
			return fmt.Errorf("failed to clone template: %w", err)
		}
	}
	return nil
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

// RunInitProject creates a basic project structure and makefile.
func RunInitProject(projectPath, projectName, sshTarget string) error {
	if sshTarget == "" {
		sshTarget = DefaultSshTarget
	}
	if projectName == "" {
		return fmt.Errorf("project name must not be empty")
	}
	projectDir := filepath.Join(projectPath, projectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	composePath := filepath.Join(projectDir, DefaultComposeFileName)
	if _, err := os.Stat(composePath); err == nil {
		return fmt.Errorf("compose file already exists at %s", composePath)
	} else if !os.IsNotExist(err) {
		return err
	}
	compose := Project{Name: projectName, Services: map[string]Service{}}
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
	if sshTarget == "" {
		sshTarget = DefaultSshTarget
	}
	user, host, err := parseSshTarget(sshTarget)
	if err != nil {
		return err
	}
	composeFilename := filepath.Base(composePath)
	dir := filepath.Dir(composePath)
	template := string(configs.MakefileTemplate)
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "COMPOSE_FILE"):
			lines[i] = fmt.Sprintf("COMPOSE_FILE ?= %s", composeFilename)
		case strings.HasPrefix(line, "REMOTE_HOST"):
			lines[i] = fmt.Sprintf("REMOTE_HOST  := %s", host)
		case strings.HasPrefix(line, "REMOTE_USER"):
			lines[i] = fmt.Sprintf("REMOTE_USER  := %s", user)
		}
	}
	result := strings.Join(lines, "\n")
	return os.WriteFile(filepath.Join(dir, "Makefile"), []byte(result), 0644)
}

// insertService prevents duplicates.
func insertService(p *types.Project, svc types.ServiceConfig) error {
	if p.Services == nil {
		p.Services = types.Services{}
	}
	if _, exists := p.Services[svc.Name]; exists {
		return fmt.Errorf("service %q already exists", svc.Name)
	}
	p.Services[svc.Name] = svc
	return nil
}

// relativisePaths converts absolute build/volume paths to relative.
func relativisePaths(p *types.Project) {
	base := p.WorkingDir
	for name, svc := range p.Services {
		if svc.Build != nil && svc.Build.Context != "" && filepath.IsAbs(svc.Build.Context) {
			if rel, err := filepath.Rel(base, svc.Build.Context); err == nil {
				svc.Build.Context = "./" + filepath.ToSlash(rel)
			}
		}
		for i := range svc.Volumes {
			v := svc.Volumes[i]
			if v.Type == "bind" && filepath.IsAbs(v.Source) {
				if rel, err := filepath.Rel(base, v.Source); err == nil {
					v.Source = "./" + filepath.ToSlash(rel)
					svc.Volumes[i] = v
				}
			}
		}
		p.Services[name] = svc
	}
}
