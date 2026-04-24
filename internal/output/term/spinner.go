package term

import (
	"fmt"
	"io"
	"os"
	"time"
)

type Spinner struct {
	stop chan struct{}
	done chan struct{}
}

// StartSpinner starts a spinner writing to w with the given message.
// Returns a no-op Spinner if w is not a TTY.
func StartSpinner(w io.Writer, message string) *Spinner {
	s := &Spinner{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}

	if !IsTTY(w) {
		close(s.done)
		return s
	}

	frames := []string{"◜", "◝", "◞", "◟"}
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for i := 0; ; i++ {
			_, _ = fmt.Fprintf(w, "\r%s %s", frames[i%len(frames)], message)
			select {
			case <-s.stop:
				_, _ = fmt.Fprintf(w, "\r\033[K")
				return
			case <-ticker.C:
			}
		}
	}()

	return s
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	<-s.done
}

func WithSpinner(outputFormat Format, msg string, fn func() error) error {
	if outputFormat == Plain {
		s := StartSpinner(os.Stderr, msg)
		defer s.Stop()
	}
	return fn()
}
