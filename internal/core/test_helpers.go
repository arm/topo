package core

import (
	"bytes"
	"io"
	"os"
)

// captureOutput captures stdout produced during f and returns it as string.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}
