package main

import (
	"os"

	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:    "describe",
	Short:  "Describe the hardware characteristics of the target",
	Long:   "Print a description of the hardware characteristics of the target, including CPU ISA features and remoteproc capabilities.",
	Hidden: true,
	Args:   cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		r := runner.For(ssh.NewDestination(targetArg))
		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()
		hwProfile, err := probe.Hardware(ctx, r)
		if err != nil {
			return err
		}

		toPrint := views.TargetDescription{HardwareProfile: hwProfile}
		return views.Print(toPrint, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(describeCmd)
	addTimeoutFlag(describeCmd, defaultTimeout)
	rootCmd.AddCommand(describeCmd)
}
