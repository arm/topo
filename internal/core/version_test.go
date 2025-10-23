package core

import (
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPrintVersion(t *testing.T) {
	out := testutil.CaptureOutput(PrintVersion)
	expected := strings.TrimSpace(VersionTxt) + "\n"
	assert.Equal(t, expected, out)
}
