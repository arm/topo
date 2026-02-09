package main

import (
	"github.com/spf13/cobra"
)

var (
	describeTarget string
)

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

		// TODO gather report

		// TODO create printable health report

		// TODO print report into yaml file

		// TODO print completion message to stdout or err
	},
}

func init() {
	addTargetFlag(describeCmd, &describeTarget)
	rootCmd.AddCommand(describeCmd)
}
