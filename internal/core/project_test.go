package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRunInitProject(t *testing.T) {
	dir := t.TempDir()
	if err := RunInitProject(dir, "proj", DefaultSshTarget); err != nil {
		t.Fatalf("RunInitProject: %v", err)
	}
	composeFile := filepath.Join(dir, "proj", DefaultComposeFileName)
	data, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("read compose: %v", err)
	}
	var p Project
	if err := yaml.Unmarshal(data, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Name != "proj" {
		t.Fatalf("expected name proj")
	}
}

func TestAddServiceWithNoRuntime(t *testing.T) {
	dir := t.TempDir()
	compose := `name: example-project
services:
  ambient-zephyr:
    build:
      context: ./ambient-zephyr
    runtime: io.containerd.remoteproc.v1
    annotations:
      remoteproc.mcu: imx-rproc
`
	composePath := filepath.Join(dir, DefaultComposeFileName)
	os.WriteFile(composePath, []byte(compose), 0644)
	var calls []struct{ URL, Dest string }
	mockCloner := func(url, dest string) error { calls = append(calls, struct{ URL, Dest string }{url, dest}); return nil }
	if err := RunAddService(composePath, "cortexa-welcome", "test", mockCloner); err != nil {
		t.Fatalf("RunAddService: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 clone call")
	}
	if calls[0].URL != "https://github.com/Arm-Debug/topo-cortexa-welcome" {
		t.Errorf("unexpected clone URL %s", calls[0].URL)
	}
}

func TestRunRemoveService(t *testing.T) {
	dir := t.TempDir()
	compose := `name: example-project
services:
  removeMe:
    build:
      context: ./removeMe
`
	composePath := filepath.Join(dir, DefaultComposeFileName)
	os.WriteFile(composePath, []byte(compose), 0644)
	if err := RunRemoveService(composePath, "removeMe"); err != nil {
		t.Fatalf("RunRemoveService: %v", err)
	}
	data, _ := os.ReadFile(composePath)
	if strings.Contains(string(data), "removeMe") {
		t.Fatalf("service still present")
	}
}

func TestGenerateMakefile(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "compose.topo.yaml")
	os.WriteFile(composePath, []byte("name: test"), 0644)
	if err := GenerateMakefile(composePath, DefaultSshTarget); err != nil {
		t.Fatalf("GenerateMakefile: %v", err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if !strings.Contains(string(content), "COMPOSE_FILE ?= compose.topo.yaml") {
		t.Fatalf("compose file line missing")
	}
}
