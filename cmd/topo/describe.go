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
	"gopkg.in/yaml.v3"
)

var describeTarget string

const targetDescriptionFilename = "target-description.yaml"

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe the hardware characteristics of the target host (CPU architecture, ISA features, etc.)",
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

		err = writeYamlFile(targetDescriptionFilename, report)
		if err != nil {
			return err
		}

		c := console.NewLogger(os.Stderr, term.Plain)
		c.Log(logger.Entry{
			Level:   logger.Info,
			Message: fmt.Sprintf("Target description written to %s", targetDescriptionFilename),
		})

		return nil
	},
}

func init() {
	addTargetFlag(describeCmd, &describeTarget)
	rootCmd.AddCommand(describeCmd)
}

func writeYamlFile(filename string, data any) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	bytes, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(wd, filename), bytes, 0644)
}
