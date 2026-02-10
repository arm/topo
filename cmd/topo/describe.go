package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/internal/describe"
	"github.com/arm-debug/topo-cli/internal/output/console"
	"github.com/arm-debug/topo-cli/internal/output/logger"
	"github.com/arm-debug/topo-cli/internal/output/term"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

var describeTarget string

const targetDescriptionFilename = "target-description.yaml"

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe the hardware characteristics of the target host including CPU ISA features and remoteproc capabilities",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		sshTarget, err := resolveTarget(describeTarget)
		if err != nil {
			return err
		}

		report, err := describe.Generate(sshTarget)
		if err != nil {
			return err
		}

		reportBytes, err := yaml.Marshal(report)
		if err != nil {
			return err
		}

		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		outputFile := filepath.Join(wd, targetDescriptionFilename)
		err = os.WriteFile(outputFile, reportBytes, 0o0644)
		if err != nil {
			return err
		}

		c := console.NewLogger(os.Stderr, term.Plain)
		c.Log(logger.Entry{
			Level:   logger.Info,
			Message: fmt.Sprintf("Target description written to %s", outputFile),
		})

		return nil
	},
}

func init() {
	addTargetFlag(describeCmd, &describeTarget)
	rootCmd.AddCommand(describeCmd)
}
