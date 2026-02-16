package main

import (
	"fmt"
	"os"

	"github.com/arm-debug/topo-cli/internal/describe"
	"github.com/arm-debug/topo-cli/internal/output/console"
	"github.com/arm-debug/topo-cli/internal/output/logger"
	"github.com/arm-debug/topo-cli/internal/output/term"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/arm-debug/topo-cli/internal/target"
	"github.com/spf13/cobra"
)

var describeTarget string

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe the hardware characteristics of the target host",
	Long:  fmt.Sprintf(`Generates a %s file that describes the hardware characteristics of the target host including CPU ISA features and remoteproc capabilities`, describe.TargetDescriptionFilename),
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		sshTarget, err := resolveTarget(describeTarget)
		if err != nil {
			return err
		}

		conn := target.NewConnection(sshTarget, ssh.ExecSSH)
		report, err := describe.GenerateTargetDescription(conn)
		if err != nil {
			return err
		}

		workDir, err := os.Getwd()
		if err != nil {
			return err
		}

		outputPath, err := describe.WriteTargetDescriptionToFile(workDir, report)
		if err != nil {
			return err
		}

		c := console.NewLogger(os.Stderr, term.Plain)
		c.Log(logger.Entry{
			Level:   logger.Info,
			Message: fmt.Sprintf("Target description written to %s", outputPath),
		})

		return nil
	},
}

func init() {
	addTargetFlag(describeCmd, &describeTarget)
	rootCmd.AddCommand(describeCmd)
}
