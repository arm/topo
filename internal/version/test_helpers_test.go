package version_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
