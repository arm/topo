package main

import (
	"os"

	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/vscode"
	"github.com/spf13/cobra"
)

var getProjectCmd = &cobra.Command{
	Use:    "get-project <compose-filepath>",
	Short:  "Print the project as JSON",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		return vscode.PrintProject(os.Stdout, composeFilePath)
	},
}

var describeCmd = &cobra.Command{
	Use:    "describe",
	Short:  "Describe the hardware characteristics of the target host",
	Long:   `Prints a description of the hardware characteristics of the target host including CPU ISA features and remoteproc capabilities`,
	Hidden: true,
	Args:   cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		r := runner.For(ssh.NewDestination(targetArg), runner.SSHOptions{Multiplex: true})
		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()
		hwProfile, err := target.ProbeHardware(ctx, r)
		if err != nil {
			return err
		}

		toPrint := templates.PrintableTargetDescription{HardwareProfile: hwProfile}
		return printable.Print(toPrint, os.Stdout, outputFormat)
	},
}

func init() {
	rootCmd.AddCommand(getProjectCmd)
	addTargetFlag(describeCmd)
	addTimeoutFlag(describeCmd, defaultTimeout)
	rootCmd.AddCommand(describeCmd)
}
