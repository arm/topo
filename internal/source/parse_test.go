package source_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("template source", func(t *testing.T) {
		got, err := source.Parse("template:hello")

		require.NoError(t, err)
		want := source.TemplateId("hello")
		assert.Equal(t, want, got)
	})

	t.Run("dir source", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  source.Dir
		}{
			{
				name:  "absolute path",
				input: "dir:/path/to/template",
				want:  source.Dir{Path: "/path/to/template"},
			},
			{
				name:  "relative path",
				input: "dir:./local/template",
				want:  source.Dir{Path: "./local/template"},
			},
			{
				name:  "path with spaces",
				input: "dir:/path/with spaces/template",
				want:  source.Dir{Path: "/path/with spaces/template"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := source.Parse(tt.input)

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("git source", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  source.Git
		}{
			{
				name:  "HTTPS without ref",
				input: "git:https://github.com/user/repo.git",
				want: source.Git{
					URL: "https://github.com/user/repo.git",
					Ref: "",
				},
			},
			{
				name:  "HTTPS with # ref",
				input: "git:https://github.com/user/repo.git#develop",
				want: source.Git{
					URL: "https://github.com/user/repo.git",
					Ref: "develop",
				},
			},
			{
				name:  "SSH without ref",
				input: "git:git@github.com:user/repo.git",
				want: source.Git{
					URL: "git@github.com:user/repo.git",
					Ref: "",
				},
			},
			{
				name:  "SSH with # ref",
				input: "git:git@github.com:user/repo.git#main",
				want: source.Git{
					URL: "git@github.com:user/repo.git",
					Ref: "main",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := source.Parse(tt.input)

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name          string
			input         string
			errorContains string
		}{
			{
				name:          "missing colon",
				input:         "template-ubuntu",
				errorContains: "invalid source format",
			},
			{
				name:          "empty value",
				input:         "template:",
				errorContains: "source value cannot be empty",
			},
			{
				name:          "empty source",
				input:         "",
				errorContains: "invalid source format",
			},
			{
				name:          "unsupported source type",
				input:         "foo:value",
				errorContains: "unsupported source type: foo",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := source.Parse(tt.input)

				assert.ErrorContains(t, err, tt.errorContains)
			})
		}
	})
}
