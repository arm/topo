package views

import (
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
)

func TestHealthStatusFormatter(t *testing.T) {
	tests := []struct {
		name   string
		status health.CheckStatus
		label  string
		color  string
	}{
		{name: "success", status: health.CheckStatusOK, label: " ✓ ", color: term.Green},
		{name: "error", status: health.CheckStatusError, label: " ✗ ", color: term.Red},
		{name: "warning", status: health.CheckStatusWarning, label: " ! ", color: term.Yellow},
		{name: "info", status: health.CheckStatusInfo, label: " i ", color: term.Blue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.label, healthStatusFormatter(false)(tt.status))
			assert.Equal(t, term.Color(tt.color, tt.label), healthStatusFormatter(true)(tt.status))
		})
	}
}
