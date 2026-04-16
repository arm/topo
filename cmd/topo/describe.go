package main

import (
	"os"

	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe the hardware characteristics of the target host",
	Long:  `Prints a description of the hardware characteristics of the target host including CPU ISA features and remoteproc capabilities`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		r := runner.For(ssh.NewDestination(targetArg), runner.SSHOptions{Multiplex: true})
		probe := target.NewHardwareProbe(r)
		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()
		hwProfile, err := probe.Probe(ctx)
		if err != nil {
			return err
		}

		toPrint := templates.PrintableTargetDescription{HardwareProfile: hwProfile}
		return printable.Print(toPrint, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(describeCmd)
	addTimeoutFlag(describeCmd, defaultTimeout)
	rootCmd.AddCommand(describeCmd)
}
