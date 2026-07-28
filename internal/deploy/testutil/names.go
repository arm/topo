package testutil

import (
	"testing"

	gtestutil "github.com/arm/topo/internal/testutil"
)

func TestImageName(t *testing.T) string {
	return "test-image-" + gtestutil.SanitiseTestName(t)
}

func TestContainerName(t *testing.T) string {
	return "test-container-" + gtestutil.SanitiseTestName(t)
}

func TestProjectName(t *testing.T) string {
	return "test-project-" + gtestutil.SanitiseTestName(t)
}
