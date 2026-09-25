// This is a throwaway, inline prompt experiment. It never reads or writes project configuration.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter/prototype/internal/demo"
	terminal "golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	layout, err := demo.Options()
	if err != nil {
		return err
	}
	if err := demo.Banner("native", layout); err != nil {
		return err
	}
	editor := newEditor(layout)
	if err := runTerminal(editor); err != nil {
		return err
	}
	return demo.Report(editor.parameters, editor.answers)
}

func runTerminal(editor *editor) (err error) {
	restoreOutput, err := prepareOutput()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, restoreOutput()) }()
	state, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	renderer := &renderer{output: os.Stderr}
	defer func() {
		clearErr := renderer.clear()
		_, outputErr := io.WriteString(os.Stderr, "\x1b[?2004l\x1b[?25h")
		err = errors.Join(err, clearErr, outputErr, terminal.Restore(int(os.Stdin.Fd()), state))
	}()
	if _, err := io.WriteString(os.Stderr, "\x1b[?2004h"); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	reads := make(chan inputRead)
	go readInput(os.Stdin, reads, ctx.Done())
	return eventLoop(ctx, editor, renderer, reads)
}

func eventLoop(ctx context.Context, editor *editor, renderer *renderer, reads <-chan inputRead) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var pending []byte
	var pendingSince time.Time
	var readErr error
	width, height := 0, 0
	dirty := true
	for {
		newWidth, newHeight, err := terminal.GetSize(int(os.Stderr.Fd()))
		if err != nil {
			return err
		}
		if newWidth < 30 || newHeight < 10 {
			return errors.New("native prototype needs at least 30 columns and 10 rows")
		}
		if dirty || newWidth != width || newHeight != height {
			width, height = newWidth, newHeight
			if err := renderer.draw(editor.view(width, height, term.NewPaletteFor(os.Stderr))); err != nil {
				return err
			}
			dirty = false
		}
		select {
		case <-ctx.Done():
			return demo.ErrCanceled
		case read := <-reads:
			if len(pending) == 0 {
				pendingSince = time.Now()
			}
			pending = append(pending, read.data...)
			readErr = read.err
		case <-ticker.C:
		}
		if len(pending) > 1<<20 {
			return errors.New("prototype paste limit exceeded; no updates saved")
		}
		done, changed, rest, err := processInput(editor, renderer, pending, time.Since(pendingSince) > 80*time.Millisecond)
		pending, dirty = rest, changed
		if err != nil || done {
			return err
		}
		if readErr != nil {
			return fmt.Errorf("input ended; no updates saved: %w", readErr)
		}
	}
}

func processInput(editor *editor, renderer *renderer, pending []byte, expired bool) (bool, bool, []byte, error) {
	changed := false
	for len(pending) > 0 {
		event, consumed := decodeInput(pending, expired)
		if consumed == 0 {
			break
		}
		pending = pending[consumed:]
		previous := editor.active
		done, err := editor.update(event)
		if err != nil {
			return false, changed, pending, err
		}
		changed = true
		if editor.layout == "sequential" && editor.active > previous {
			if err := renderer.clear(); err != nil {
				return false, changed, pending, err
			}
			_, err := fmt.Fprint(os.Stderr, demo.Transcript(editor.parameters[previous], editor.answers[previous])+"\r\n")
			if err != nil {
				return false, changed, pending, err
			}
		}
		if done {
			return true, changed, pending, nil
		}
	}
	return false, changed, pending, nil
}
