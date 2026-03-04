package main

import (
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install components to target",
}

func init() {
	rootCmd.AddCommand(installCmd)
}
