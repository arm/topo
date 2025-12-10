package testutil

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const TestSshTarget = "test-target"

// captureOutput captures stdout produced during f and returns it as string.
func CaptureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

func RequireDocker(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found. Install Docker: https://docs.docker.com/desktop/")
	}
}

func RequireWriteFile(t testing.TB, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
}

func SanitiseTestName(t testing.TB) string {
	name := strings.ToLower(t.Name())
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ",", "")
	return name
}

func WriteComposeFile(t *testing.T, dir, content string) string {
	t.Helper()
	composePath := filepath.Join(dir, template.ComposeFilename)
	RequireWriteFile(t, composePath, content)
	return composePath
}

func ParseYAMLString(s string) (*yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(s), &root); err != nil {
		return nil, err
	}
	doc := root.Content[0]
	return doc, nil
}
