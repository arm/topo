package term

import (
	"fmt"
	"io"
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
		for i := 0; ; i++ {
			select {
			case <-s.stop:
				fmt.Fprintf(w, "\r\033[K")
				return
			default:
				fmt.Fprintf(w, "\r%s %s", frames[i%len(frames)], message)
				time.Sleep(100 * time.Millisecond)
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
