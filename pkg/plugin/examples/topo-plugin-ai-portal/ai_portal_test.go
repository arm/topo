package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPortalURL(t *testing.T) {
	t.Run("encodes project name once and replaces stale names", func(t *testing.T) {
		baseURL := "https://portal.example/models?keep=yes&topoProjectName=old&topoProjectName=older"
		callbackURL := "http://127.0.0.1:1234/callback?token=a+b"
		filters := ModelQuery{ProjectName: " \t模型 + demo:  50% & /?# %2B\u2003", Task: "image-classification"}

		got, err := buildPortalURL(baseURL, callbackURL, filters)

		require.NoError(t, err)
		address, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, url.Values{
			"keep":            {"yes"},
			"task":            {"image-classification"},
			"topoCallback":    {callbackURL},
			"topoProjectName": {"模型 + demo:  50% & /?# %2B"},
		}, address.Query())
		assert.Contains(t, address.RawQuery,
			"topoProjectName=%E6%A8%A1%E5%9E%8B+%2B+demo%3A++50%25+%26+%2F%3F%23+%252B")
	})

	t.Run("omits missing and blank names and removes stale names", func(t *testing.T) {
		for _, test := range []struct {
			name        string
			baseURL     string
			projectName string
		}{
			{name: "missing", baseURL: "https://portal.example/models"},
			{
				name:    "missing replaces stale values",
				baseURL: "https://portal.example/models?topoProjectName=old&topoProjectName=older",
			},
			{
				name:        "blank replaces stale values",
				baseURL:     "https://portal.example/models?topoProjectName=old&topoProjectName=older",
				projectName: " \t\n\u2003",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				callbackURL := "http://127.0.0.1:1234/callback"
				filters := ModelQuery{ProjectName: test.projectName}

				got, err := buildPortalURL(test.baseURL, callbackURL, filters)

				require.NoError(t, err)
				address, err := url.Parse(got)
				require.NoError(t, err)
				assert.Equal(t, url.Values{"topoCallback": {callbackURL}}, address.Query())
			})
		}
	})
}
