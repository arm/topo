package fact_test

import (
	"testing"

	"github.com/arm/topo/internal/fact"
	"github.com/stretchr/testify/assert"
)

func TestValue(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		for _, tc := range []struct {
			value fact.Value
			want  string
		}{
			{fact.Value{Type: "boolean"}, "false"},
			{fact.Value{Type: "boolean", Boolean: true}, "true"},
			{fact.Value{Type: "integer"}, "0"},
			{fact.Value{Type: "integer", Integer: -42}, "-42"},
			{fact.Value{Type: "string"}, ""},
			{fact.Value{Type: "string", Text: "example"}, "example"},
		} {
			got := tc.value.String()

			assert.Equal(t, tc.want, got)
		}
	})

	t.Run("String panics for an unsupported type", func(t *testing.T) {
		value := fact.Value{Type: "unknown"}

		assert.PanicsWithValue(t, `unsupported value type: "unknown"`, func() {
			_ = value.String()
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Run("accepts typed scalars including zero values", func(t *testing.T) {
			for _, value := range []fact.Value{
				{Type: "boolean"},
				{Type: "boolean", Boolean: true},
				{Type: "integer"},
				{Type: "integer", Integer: -1},
				{Type: "string"},
				{Type: "string", Text: "example"},
			} {
				err := value.Validate()

				assert.NoError(t, err, "%+v", value)
			}
		})

		t.Run("rejects unknown types and conflicting fields", func(t *testing.T) {
			for _, value := range []fact.Value{
				{},
				{Type: "array"},
				{Type: "boolean", Integer: 1},
				{Type: "boolean", Text: "false"},
				{Type: "integer", Boolean: true},
				{Type: "integer", Text: "1"},
				{Type: "string", Boolean: true},
				{Type: "string", Integer: 1},
			} {
				err := value.Validate()

				assert.Error(t, err, "%+v", value)
			}
		})
	})
}

func TestReport(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		t.Run("accepts empty reports and independently scoped facts", func(t *testing.T) {
			sample := fact.Fact{Key: "example", Value: fact.Value{Type: "boolean"}}
			for _, report := range []fact.Report{
				{},
				{Groups: []fact.Group{
					{ID: "empty"},
					{ID: "a", Facts: []fact.Fact{sample}},
					{ID: "b", Facts: []fact.Fact{sample}},
				}},
			} {
				err := report.Validate()

				assert.NoError(t, err)
			}
		})

		for _, test := range []struct {
			name   string
			groups []fact.Group
		}{
			{"rejects empty group IDs", []fact.Group{{}}},
			{"rejects duplicate group IDs", []fact.Group{{ID: "a"}, {ID: "a"}}},
			{"rejects empty fact keys", []fact.Group{{ID: "a", Facts: []fact.Fact{{Value: fact.Value{Type: "boolean"}}}}}},
			{"rejects duplicate fact keys", []fact.Group{{ID: "a", Facts: []fact.Fact{
				{Key: "example", Value: fact.Value{Type: "boolean"}},
				{Key: "example", Value: fact.Value{Type: "boolean"}},
			}}}},
			{"rejects invalid values", []fact.Group{{ID: "a", Facts: []fact.Fact{{Key: "example"}}}}},
		} {
			t.Run(test.name, func(t *testing.T) {
				report := fact.Report{Groups: test.groups}

				err := report.Validate()

				assert.Error(t, err)
			})
		}
	})

	t.Run("Find", func(t *testing.T) {
		t.Run("returns the fact matching both group ID and key", func(t *testing.T) {
			want := fact.Fact{Key: "key-a", Value: fact.Value{Type: "integer", Integer: 1}}
			report := fact.Report{Groups: []fact.Group{
				{ID: "group-a", Facts: []fact.Fact{{Key: "key-a", Value: fact.Value{Type: "integer", Integer: 2}}}},
				{ID: "group-b", Facts: []fact.Fact{{Key: "key-b"}, want}},
			}}

			got, ok := report.Find("group-b", "key-a")

			assert.True(t, ok)
			assert.Equal(t, want, got)
		})

		t.Run("returns not found when the key does not exist in the specified group", func(t *testing.T) {
			report := fact.Report{Groups: []fact.Group{
				{ID: "group-a", Facts: []fact.Fact{{Key: "key-a"}}},
				{ID: "group-b", Facts: []fact.Fact{{Key: "key-b"}}},
			}}

			got, ok := report.Find("group-a", "key-b")

			assert.False(t, ok)
			assert.Empty(t, got)
		})
	})
}
