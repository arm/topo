package version_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const homebrewFormula = `class Topo < Formula
  desc "Compose, parameterize, and deploy containerized examples for Arm hardware"
  version "5.1.1"
end`

func TestFetchLatestHomebrew(t *testing.T) {
	t.Run("returns version from Homebrew formula", func(t *testing.T) {
		srv := createTestServerWithBody(t, homebrewFormula)

		got, err := version.FetchLatestHomebrew(context.Background(), srv.URL)

		require.NoError(t, err)
		assert.Equal(t, "5.1.1", got)
	})
}

func TestParseHomebrewFormulaVersion(t *testing.T) {
	t.Run("returns version from formula", func(t *testing.T) {
		got, err := version.ParseHomebrewFormulaVersion(homebrewFormula)

		require.NoError(t, err)
		assert.Equal(t, "5.1.1", got)
	})
}
