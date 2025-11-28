package main

import (
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services in compose files",
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}
