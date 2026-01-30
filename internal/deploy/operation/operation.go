package operation

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Operation interface {
	Description() string
	Run(cmdOutput io.Writer) error
	DryRun(output io.Writer) error
}

// SetupExitCleanup sets up a handler to run an operation once when the program exits due to an interrupt signal.
func SetupExitCleanup(operation Operation, w io.Writer, exit func(int)) func() {
	var once sync.Once
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	doCleanupOnce := func() {
		once.Do(func() {
			if operation != nil {
				if err := operation.Run(w); err != nil {
					_, _ = fmt.Fprintf(w, "WARN: failed to cleanup on exit: %v\n", err)
				}
			}
			signal.Stop(sigChan)
			close(sigChan)
		})
	}
	go func() {
		sig, ok := <-sigChan
		if !ok || sig == nil {
			return
		}
		doCleanupOnce()
		exit(1)
	}()

	return doCleanupOnce
}
