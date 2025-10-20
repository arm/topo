package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintVersion(t *testing.T) {
	out := captureOutput(PrintVersion)
	expected := strings.TrimSpace(VersionTxt) + "\n"
	assert.Equal(t, expected, out)
}
