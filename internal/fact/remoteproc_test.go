package fact_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteprocProvider(t *testing.T) {
	const probe = `sh -c 'LC_ALL=C grep -H ^ /sys/class/remoteproc/*/name'`

	t.Run("reports distinct devices with duplicate names and gaps in numbering", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			probe: {
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
			probe: {Err: assert.AnError, Stderr: "grep: /sys/class/remoteproc/*/name: No such file or directory\n"},
		}}

		got, err := (fact.RemoteprocProvider{}).Facts(context.Background(), r)

		require.NoError(t, err)
		assert.Equal(t, fact.Report{}, got)
	})

	t.Run("rejects an empty name", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			probe: {Output: "/sys/class/remoteproc/remoteproc2/name: "},
		}}

		_, err := (fact.RemoteprocProvider{}).Facts(context.Background(), r)

		assert.ErrorContains(t, err, "empty remoteproc name")
	})

	t.Run("propagates probe errors", func(t *testing.T) {
		r := &runner.Fake{Commands: map[string]runner.FakeResult{
			probe: {Err: assert.AnError, Stderr: "Permission denied"},
		}}

		_, err := (fact.RemoteprocProvider{}).Facts(context.Background(), r)

		assert.ErrorIs(t, err, assert.AnError)
	})
}
