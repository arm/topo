package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetProject(t *testing.T) {
	dir := t.TempDir()
	compose := `name: demo
services: {}`
	composePath := filepath.Join(dir, DefaultComposeFileName)
	os.WriteFile(composePath, []byte(compose), 0644)
	out := captureOutput(func() { GetProject(composePath) })
	if !strings.Contains(out, "\"name\": \"demo\"") {
		t.Fatalf("missing name in output: %s", out)
	}
}

func TestGetConfigMetadata(t *testing.T) {
	out := captureOutput(func() {
		if err := GetConfigMetadata(); err != nil {
			t.Fatalf("err: %v", err)
		}
	})
	if !strings.Contains(out, "boards") {
		t.Fatalf("expected boards field")
	}
}
