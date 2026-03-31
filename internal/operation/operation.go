package operation

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/arm/topo/internal/output/logger"
)

type Operation interface {
	Description() string
	Run(cmdOutput io.Writer) error
}

// SetupExitCleanup sets up a handler to run an operation once when the program exits due to an interrupt signal.
func SetupExitCleanup(w io.Writer, operation Operation, exit func(int)) func() {
	var once sync.Once
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	doCleanupOnce := func() {
		once.Do(func() {
			if operation != nil {
				if err := operation.Run(w); err != nil {
					logger.Warn(fmt.Sprintf(": failed to cleanup on exit: %v\n", err))
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
