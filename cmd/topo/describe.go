package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/describe"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe the hardware characteristics of the target host",
	Long:  fmt.Sprintf(`Generates a %s file that describes the hardware characteristics of the target host including CPU ISA features and remoteproc capabilities`, describe.TargetDescriptionFilename),
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		r := runner.For(ssh.NewDestination(targetArg), runner.SSHOptions{Multiplex: true, ConnectTimeout: sshConnectTimeout})
		probe := target.NewHardwareProbe(r)
		hwProfile, err := probe.Probe()
		if err != nil {
			return err
		}

		workDir, err := os.Getwd()
		if err != nil {
			return err
		}

		outputPath, err := describe.WriteTargetDescriptionToFile(workDir, hwProfile)
		if err != nil {
			return err
		}

		logger.Info(fmt.Sprintf("Target description written to %s", outputPath))

		return nil
	},
}

func init() {
	addTargetFlag(describeCmd)
	rootCmd.AddCommand(describeCmd)
}
