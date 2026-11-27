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
		t.Run("accepts empty reports and valid facts", func(t *testing.T) {
			for _, report := range []fact.Report{
				nil,
				{"example": {Value: fact.Value{Type: "boolean"}}},
			} {
				err := report.Validate()

				assert.NoError(t, err)
			}
		})

		for _, test := range []struct {
			name   string
			report fact.Report
		}{
			{"rejects empty fact keys", fact.Report{"": {Value: fact.Value{Type: "boolean"}}}},
			{"rejects invalid values", fact.Report{"example": {}}},
		} {
			t.Run(test.name, func(t *testing.T) {
				err := test.report.Validate()

				assert.Error(t, err)
			})
		}
	})
}
