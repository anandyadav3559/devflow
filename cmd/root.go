package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devflow",
	Short: "A command orchestrator for developers",
	Long: `DevFlow is a local development environment orchestrator.

It lets you define multi-service workflows in YAML and run them with 
dependency management, logging, and easy reuse.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
