package version_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arm/topo/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const artifactoryHTML = `<!DOCTYPE html>
<html>
<head><meta name="robots" content="noindex" />
<title>Index of devx-topo/</title>
</head>
<body>
<h1>Index of devx-topo/</h1>
<pre>Name                        Last modified      Size</pre><hr/>
<pre><a href="v0.0.0-2026-03-11-15-52-41/">v0.0.0-2026-03-11-15-52-41/</a>  11-Mar-2026 16:00    -
<a href="v0.0.0-2026-03-12-09-42-30/">v0.0.0-2026-03-12-09-42-30/</a>  12-Mar-2026 09:50    -
<a href="v1.0.0/">v1.0.0/</a>                      13-Mar-2026 08:38    -
<a href="v1.0.0-2026-03-13-12-10-31/">v1.0.0-2026-03-13-12-10-31/</a>  13-Mar-2026 12:14    -
<a href="v1.1.0/">v1.1.0/</a>                      13-Mar-2026 11:54    -
<a href="v1.2.0-2026-03-16-11-36-49/">v1.2.0-2026-03-16-11-36-49/</a>  16-Mar-2026 11:41    -
<a href="v1.3.0/">v1.3.0/</a>                      16-Mar-2026 13:10    -
<a href="v1.3.1/">v1.3.1/</a>                      17-Mar-2026 14:45    -
<a href="v1.4.0/">v1.4.0/</a>                      19-Mar-2026 10:03    -
<a href="v1.4.1/">v1.4.1/</a>                      25-Mar-2026 13:02    -
<a href="v2.0.0/">v2.0.0/</a>                      02-Apr-2026 12:18    -
<a href="v3.0.0/">v3.0.0/</a>                      07-Apr-2026 16:06    -
<a href="v3.0.1/">v3.0.1/</a>                      09-Apr-2026 10:38    -
<a href="v4.0.0/">v4.0.0/</a>                      10-Apr-2026 15:12    -
<a href="v4.1.0/">v4.1.0/</a>                      14-Apr-2026 15:56    -
</pre>
<hr/><address style="font-size:small;">Artifactory Online Server</address></body></html>`

func createTestServerWithBody(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

func TestFetchLatest(t *testing.T) {
	t.Run("can reach real artifactory index page", func(t *testing.T) {
		_, err := version.FetchLatest(context.Background(), version.ArtifactoryBaseURL)

		require.NoError(t, err)
	})

	t.Run("returns highest version from artifactory index", func(t *testing.T) {
		srv := createTestServerWithBody(t, artifactoryHTML)

		got, err := version.FetchLatest(context.Background(), srv.URL)

		require.NoError(t, err)
		assert.Equal(t, "v4.1.0", got)
	})

	t.Run("picks correct version when order is scrambled", func(t *testing.T) {
		body := `<a href="v2.0.0/">v2.0.0/</a>
<a href="v10.0.0/">v10.0.0/</a>
<a href="v1.9.0/">v1.9.0/</a>
<a href="v2.1.0/">v2.1.0/</a>`
		srv := createTestServerWithBody(t, body)

		got, err := version.FetchLatest(context.Background(), srv.URL)

		require.NoError(t, err)
		assert.Equal(t, "v10.0.0", got)
	})

	t.Run("deduplicates repeated versions", func(t *testing.T) {
		body := `<a href="v1.0.0/">v1.0.0/</a>
<a href="v1.0.0-2026-03-13/">v1.0.0-2026-03-13/</a>`
		srv := createTestServerWithBody(t, body)

		got, err := version.FetchLatest(context.Background(), srv.URL)

		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", got)
	})

	t.Run("returns error when no versions found", func(t *testing.T) {
		srv := createTestServerWithBody(t, "no versions here")

		_, err := version.FetchLatest(context.Background(), srv.URL)

		assert.ErrorContains(t, err, "no versions found")
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		_, err := version.FetchLatest(context.Background(), srv.URL)

		assert.ErrorContains(t, err, "HTTP 500")
	})

	t.Run("returns error on connection failure", func(t *testing.T) {
		_, err := version.FetchLatest(context.Background(), "something-invalid")
		assert.Error(t, err)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		srv := createTestServerWithBody(t, artifactoryHTML)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := version.FetchLatest(ctx, srv.URL)

		assert.Error(t, err)
	})
}
