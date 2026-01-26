package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePrintable struct {
	jsonStr  string
	plainStr string
	jsonErr  error
	plainErr error
}

func (f fakePrintable) AsJSON() (string, error)  { return f.jsonStr, f.jsonErr }
func (f fakePrintable) AsPlain() (string, error) { return f.plainStr, f.plainErr }

func TestPrintable(t *testing.T) {
	t.Run("AsPlain", func(t *testing.T) {
		t.Run("prints plain output when no error", func(t *testing.T) {
			var buf bytes.Buffer
			p := output.NewPrinter(&buf, output.PlainFormat)
			fp := fakePrintable{plainStr: "hello-plain"}

			err := p.Print(fp)

			require.NoError(t, err)
			assert.Equal(t, "hello-plain", buf.String())
		})

		t.Run("propagates error", func(t *testing.T) {
			var buf bytes.Buffer
			p := output.NewPrinter(&buf, output.PlainFormat)
			want := errors.New("plain failed")
			fp := fakePrintable{plainErr: want}

			got := p.Print(fp)

			require.Error(t, got)
			assert.Equal(t, want, got)
			assert.Equal(t, "", buf.String())
		})
	})

	t.Run("AsJSON", func(t *testing.T) {
		t.Run("prints json output when no error", func(t *testing.T) {
			var buf bytes.Buffer
			p := output.NewPrinter(&buf, output.JSONFormat)
			fp := fakePrintable{jsonStr: `{"k":"v"}`}

			err := p.Print(fp)

			require.NoError(t, err)
			assert.Equal(t, `{"k":"v"}`+"\n", buf.String())
		})

		t.Run("propagates error", func(t *testing.T) {
			var buf bytes.Buffer
			p := output.NewPrinter(&buf, output.JSONFormat)
			want := errors.New("json failed")
			fp := fakePrintable{jsonErr: want}

			got := p.Print(fp)

			require.Error(t, got)
			assert.Equal(t, want, got)
			assert.Equal(t, "", buf.String())
		})
	})
}

func TestPrintTemplateRepos(t *testing.T) {
	t.Run("prints multiple items correctly", func(t *testing.T) {
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Url:         "url.git",
				Ref:         "main",
			},
			{
				Id:          "name-of-other-project",
				Description: "blah blah blah",
				Url:         "url.git",
				Ref:         "main",
			},
		}

		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, output.PlainFormat)

		err := output.PrintTemplateRepos(printer, repos)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  blah blah blah

name-of-other-project | url.git | main
  blah blah blah

`
		assert.Equal(t, want, buf.String())
	})

	t.Run("ignores features when none present", func(t *testing.T) {
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Url:         "url.git",
				Ref:         "main",
			},
		}

		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, output.PlainFormat)

		err := output.PrintTemplateRepos(printer, repos)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  blah blah blah

`
		assert.Equal(t, want, buf.String())
	})

	t.Run("includes features when present", func(t *testing.T) {
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, output.PlainFormat)

		err := output.PrintTemplateRepos(printer, repos)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  blah blah blah

`
		assert.Equal(t, want, buf.String())
	})

	t.Run("correctly wraps long descriptions", func(t *testing.T) {
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "This sentence exists purely to verify that text wrapping behaves correctly when the content is long enough to span multiple lines.",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, output.PlainFormat)

		err := output.PrintTemplateRepos(printer, repos)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  This sentence exists purely to verify that text wrapping behaves correctly
  when the content is long enough to span multiple lines.

`
		assert.Equal(t, want, buf.String())
	})

	t.Run("correctly splits paragraphs in the description", func(t *testing.T) {
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah\n\nblah blah blah",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, output.PlainFormat)

		err := output.PrintTemplateRepos(printer, repos)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  blah blah blah

  blah blah blah

`
		assert.Equal(t, want, buf.String())
	})

	t.Run("correctly prints to json", func(t *testing.T) {
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah\n\nblah blah blah",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		var buf bytes.Buffer
		printer := output.NewPrinter(&buf, output.JSONFormat)

		err := output.PrintTemplateRepos(printer, repos)
		require.NoError(t, err)

		var gotObj any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &gotObj))

		wantObj := []any{
			map[string]any{
				"id":          "name-of-project",
				"description": "blah blah blah\n\nblah blah blah",
				"features":    []any{"walnut", "almond"},
				"url":         "url.git",
				"ref":         "main",
			},
		}

		assert.Equal(t, wantObj, gotObj)
	})
}

func TestPrintHealthReport(t *testing.T) {
	t.Run("PlainFormat", func(t *testing.T) {
		t.Run("it renders the dependencies", func(t *testing.T) {
			report := health.Report{}
			report.Host.Dependencies = []health.HealthCheck{{
				Name:    "Flux Capacitor",
				Healthy: true,
			}}

			var buf bytes.Buffer
			printer := output.NewPrinter(&buf, output.PlainFormat)

			err := output.PrintHealthReport(printer, report)
			require.NoError(t, err)

			assert.Contains(t, buf.String(), "Flux Capacitor")
		})

		t.Run("it renders connection failures", func(t *testing.T) {
			report := health.Report{}
			report.Target.Connectivity = health.HealthCheck{
				Name:    "Connected",
				Healthy: false,
			}

			var buf bytes.Buffer
			printer := output.NewPrinter(&buf, output.PlainFormat)

			err := output.PrintHealthReport(printer, report)
			require.NoError(t, err)

			assert.Contains(t, buf.String(), "Connected: ❌")
		})

		t.Run("when connected it renders cpu features", func(t *testing.T) {
			report := health.Report{}
			report.Target.Connectivity = health.HealthCheck{
				Name:    "Connected",
				Healthy: true,
			}
			report.Target.Features = []string{"FOO", "BAR"}

			var buf bytes.Buffer
			printer := output.NewPrinter(&buf, output.PlainFormat)

			err := output.PrintHealthReport(printer, report)
			require.NoError(t, err)

			assert.Contains(t, buf.String(), "FOO, BAR")
		})

		t.Run("when not connected, it does not render cpu features", func(t *testing.T) {
			report := health.Report{}
			report.Target.Connectivity = health.HealthCheck{
				Name:    "Connected",
				Healthy: false,
			}

			var buf bytes.Buffer
			printer := output.NewPrinter(&buf, output.PlainFormat)

			err := output.PrintHealthReport(printer, report)
			require.NoError(t, err)

			assert.NotContains(t, buf.String(), "Features")
		})

		t.Run("when localhost, it skips connectivity check and shows features", func(t *testing.T) {
			report := health.Report{}
			report.Target.IsLocalhost = true
			report.Target.Features = []string{"FOO", "BAR"}

			var buf bytes.Buffer
			printer := output.NewPrinter(&buf, output.PlainFormat)

			err := output.PrintHealthReport(printer, report)
			require.NoError(t, err)

			assert.NotContains(t, buf.String(), "Connected")
			assert.Contains(t, buf.String(), "FOO, BAR")
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders report as valid JSON with expected fields", func(t *testing.T) {
			report := health.Report{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{Name: "Flux Capacitor", Healthy: true},
					},
				},
				Target: health.TargetReport{
					Connectivity: health.HealthCheck{Name: "Connected", Healthy: true},
				},
			}

			var buf bytes.Buffer
			printer := output.NewPrinter(&buf, output.JSONFormat)

			err := output.PrintHealthReport(printer, report)
			require.NoError(t, err)

			want := `{
				"Host": {
					"Dependencies": [
						{"Name":"Flux Capacitor","Healthy":true,"Value":""}
					]
				},
				"Target": {
					"IsLocalhost": false,
					"Connectivity": {"Name":"Connected","Healthy":true,"Value":""},
					"Dependencies": [],
					"Features": [],
					"SubsystemDriver": {"Name":"","Healthy":false,"Value":""}
				}
			}`

			assert.JSONEq(t, want, buf.String())
		})
	})
}
