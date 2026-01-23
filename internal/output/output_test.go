package output_test

import (
	"bytes"
	"errors"
	"testing"

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
