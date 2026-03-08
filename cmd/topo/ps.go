package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/arm/topo/internal/output/console"
	"github.com/arm/topo/internal/output/logger"
	ps "github.com/arm/topo/internal/topops"
	"github.com/spf13/cobra"
)

var (
	psPath            string
	psRefreshInterval time.Duration
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Expose target topology and containers as a local virtual filesystem",
	Long: `Expose target topology and containers as a local virtual filesystem.

The command creates a local topology tree (default: ./topology) and keeps it
updated by polling the selected target. Container control files support two
operations:
  - start
  - stop

Write the operation to a container's command file, for example:
  echo stop > topology/Host/my-service--abc123def456/command
`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}
		c := console.NewLogger(os.Stderr, outputFormat)

		target, err := requireTarget(cmd)
		if err != nil {
			return err
		}
		if strings.TrimSpace(psPath) == "" {
			psPath = target
		}

		if psRefreshInterval <= 0 {
			return fmt.Errorf("refresh interval must be greater than zero")
		}

		manager := ps.NewManager(target, ps.Options{
			RootPath:        psPath,
			RefreshInterval: psRefreshInterval,
		}, c)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		c.Log(logger.Entry{
			Level:   logger.Info,
			Message: fmt.Sprintf("topology view running at %q; press Ctrl+C to stop", psPath),
		})
		return manager.Run(ctx)
	},
}

func init() {
	addTargetFlag(psCmd)
	psCmd.Flags().StringVar(&psPath, "path", "", "Path where the topology virtual filesystem is created (default: target name)")
	psCmd.Flags().DurationVar(&psRefreshInterval, "refresh-interval", 2*time.Second, "How often topology is refreshed from the target")
	rootCmd.AddCommand(psCmd)
}
