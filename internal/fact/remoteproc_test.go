package fact_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteprocProvider(t *testing.T) {
	t.Run("reports distinct devices with duplicate names and gaps in numbering", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"grep -H . /sys/class/remoteproc/*/name": {
				Output: `/sys/class/remoteproc/remoteproc2/name:Cortex M4
/sys/class/remoteproc/remoteproc7/name:Cortex M4
`,
			},
		}}
		want := fact.Report{Groups: []fact.Group{
			{
				ID: "remoteproc2",
				Facts: []fact.Fact{{
					Key:         fact.FactRemoteprocName,
					Value:       fact.Value{Type: "string", Text: "Cortex M4"},
					Label:       "Remote processor name",
					Description: "Kernel-reported remote processor name",
				}},
			},
			{
				ID: "remoteproc7",
				Facts: []fact.Fact{{
					Key:         fact.FactRemoteprocName,
					Value:       fact.Value{Type: "string", Text: "Cortex M4"},
					Label:       "Remote processor name",
					Description: "Kernel-reported remote processor name",
				}},
			},
		}}

		got, err := (fact.RemoteprocProvider{}).Facts(context.Background(), r)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.NoError(t, got.Validate())
	})

	t.Run("returns empty when sysfs is unavailable", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"grep -H . /sys/class/remoteproc/*/name": {Err: errors.New("No such file or directory")},
		}}

		got, err := (fact.RemoteprocProvider{}).Facts(context.Background(), r)

		require.NoError(t, err)
		assert.Equal(t, fact.Report{}, got)
	})

	t.Run("rejects an empty name", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"grep -H . /sys/class/remoteproc/*/name": {Output: "/sys/class/remoteproc/remoteproc2/name: "},
		}}

		_, err := (fact.RemoteprocProvider{}).Facts(context.Background(), r)

		assert.ErrorContains(t, err, "empty remoteproc name")
	})

	t.Run("propagates wrapped timeout errors", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			"grep -H . /sys/class/remoteproc/*/name": {Err: fmt.Errorf("reading remoteproc: %w", runner.ErrTimeout)},
		}}

		_, err := (fact.RemoteprocProvider{}).Facts(context.Background(), r)

		assert.ErrorIs(t, err, runner.ErrTimeout)
	})
}
