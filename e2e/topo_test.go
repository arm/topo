package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/configs"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "topo")
	cmd := exec.Command("go", "build", "-o", bin, "../cmd/topo")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func TestIntegration_ListTemplates(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "list-templates")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(arr) == 0 {
		t.Fatalf("expected templates")
	}
}

func TestIntegration_Version(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != strings.TrimSpace(configs.VersionTxt) {
		t.Fatalf("version mismatch: %s vs %s", out, configs.VersionTxt)
	}
}
